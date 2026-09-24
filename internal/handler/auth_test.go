package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"todo-list-api/internal/auth"
	"todo-list-api/internal/db"

	"github.com/gin-gonic/gin"
)

type dataEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type pairPayload struct {
	User         map[string]any `json:"user"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
}

func newAuthTestRouter(t *testing.T) *gin.Engine {
	router, _ := newAuthTestRouterDB(t)
	return router
}

func newAuthTestRouterDB(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing")
	testDB := db.SetupTestDB(t)
	svc, err := auth.NewAuthService(auth.NewUserRepository(testDB), auth.NewRefreshTokenRepository(testDB))
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	router := gin.New()
	RegisterAuthRoutes(router, svc)
	return router, testDB
}

func postJSON(t *testing.T, router *gin.Engine, path string, body any) (int, json.RawMessage) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var env dataEnvelope
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil && w.Code < 500 {
			t.Fatalf("parse response (%d): %v — body: %s", w.Code, err, w.Body.String())
		}
	}
	return w.Code, env.Data
}

func registerUser(t *testing.T, router *gin.Engine, email string) pairPayload {
	t.Helper()
	code, data := postJSON(t, router, "/v1/register", map[string]string{
		"name":     "Test User",
		"email":    email,
		"password": "password123",
	})
	if code != http.StatusCreated {
		t.Fatalf("setup register: expected 201, got %d — %s", code, data)
	}
	var pair pairPayload
	if err := json.Unmarshal(data, &pair); err != nil {
		t.Fatalf("parse pair: %v", err)
	}
	return pair
}

func TestRegisterSuccess(t *testing.T) {
	router := newAuthTestRouter(t)
	code, data := postJSON(t, router, "/v1/register", map[string]string{
		"name":     "Ada Lovelace",
		"email":    "ada@example.com",
		"password": "password123",
	})
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — %s", code, data)
	}
	var pair pairPayload
	if err := json.Unmarshal(data, &pair); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected access_token and refresh_token")
	}
	if pair.User["email"] != "ada@example.com" {
		t.Fatalf("expected user email, got %v", pair.User["email"])
	}
	if _, leaked := pair.User["password_hash"]; leaked {
		t.Fatal("user DTO must not leak password_hash")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	router := newAuthTestRouter(t)
	registerUser(t, router, "dupe@example.com")
	code, _ := postJSON(t, router, "/v1/register", map[string]string{
		"name":     "Dupe",
		"email":    "dupe@example.com",
		"password": "password123",
	})
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", code)
	}
}

func TestRegisterValidationErrors(t *testing.T) {
	router := newAuthTestRouter(t)
	cases := []struct {
		name string
		body map[string]string
	}{
		{"missing fields", map[string]string{}},
		{"invalid email", map[string]string{"name": "A", "email": "not-an-email", "password": "password123"}},
		{"short password", map[string]string{"name": "A", "email": "a@example.com", "password": "short"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := postJSON(t, router, "/v1/register", tc.body)
			if code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d", code)
			}
		})
	}
}

func TestLoginSuccess(t *testing.T) {
	router := newAuthTestRouter(t)
	registerUser(t, router, "login@example.com")
	code, data := postJSON(t, router, "/v1/login", map[string]string{
		"email":    "login@example.com",
		"password": "password123",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", code, data)
	}
	var pair pairPayload
	if err := json.Unmarshal(data, &pair); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
}

func TestLoginBadCredentials(t *testing.T) {
	router := newAuthTestRouter(t)
	registerUser(t, router, "user@example.com")
	cases := []struct {
		name string
		body map[string]string
	}{
		{"unknown email", map[string]string{"email": "nobody@example.com", "password": "password123"}},
		{"wrong password", map[string]string{"email": "user@example.com", "password": "wrongpass1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _ := postJSON(t, router, "/v1/login", tc.body)
			if code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", code)
			}
		})
	}
}

func TestLoginValidationErrors(t *testing.T) {
	router := newAuthTestRouter(t)
	code, _ := postJSON(t, router, "/v1/login", map[string]string{})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", code)
	}
}

func TestRefreshSuccess(t *testing.T) {
	router := newAuthTestRouter(t)
	pair := registerUser(t, router, "refresh@example.com")
	code, data := postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", code, data)
	}
	var newPair pairPayload
	if err := json.Unmarshal(data, &newPair); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Fatal("expected new token pair")
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatal("expected rotated refresh token")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	router := newAuthTestRouter(t)
	pair := registerUser(t, router, "badrefresh@example.com")

	// rotate once, then reuse the old (now revoked) token
	code, _ := postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusOK {
		t.Fatalf("first refresh: expected 200, got %d", code)
	}
	code, _ = postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusBadRequest {
		t.Fatalf("reused token: expected 400, got %d", code)
	}

	code, _ = postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": "not-a-real-token"})
	if code != http.StatusBadRequest {
		t.Fatalf("garbage token: expected 400, got %d", code)
	}
}

func TestRefreshExpiredToken(t *testing.T) {
	router, testDB := newAuthTestRouterDB(t)
	pair := registerUser(t, router, "expired@example.com")
	if _, err := testDB.Exec(
		"UPDATE refresh_tokens SET expires_at = ? WHERE token = ?",
		time.Now().Add(-time.Hour), pair.RefreshToken,
	); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	code, _ := postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusBadRequest {
		t.Fatalf("expired token: expected 400, got %d", code)
	}
}

func TestRefreshMissingToken(t *testing.T) {
	router := newAuthTestRouter(t)
	code, _ := postJSON(t, router, "/v1/refresh", map[string]string{})
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing token, got %d", code)
	}
}

func TestLogoutSuccess(t *testing.T) {
	router := newAuthTestRouter(t)
	pair := registerUser(t, router, "logout@example.com")
	code, _ := postJSON(t, router, "/v1/logout", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}

	// reuse revoked token → 400 on both logout and refresh
	code, _ = postJSON(t, router, "/v1/logout", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusBadRequest {
		t.Fatalf("revoked logout: expected 400, got %d", code)
	}
	code, _ = postJSON(t, router, "/v1/refresh", map[string]string{"refresh_token": pair.RefreshToken})
	if code != http.StatusBadRequest {
		t.Fatalf("revoked refresh: expected 400, got %d", code)
	}
}
