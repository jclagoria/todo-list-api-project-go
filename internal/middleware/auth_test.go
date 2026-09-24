package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"todo-list-api/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testUserID      = "550e8400-e29b-41d4-a716-446655440000"
	testSecret      = "middleware-test-secret"
	testRequestID   = "req-fixed-0001"
	testPassMessage = "Missing or invalid token"
)

type stubAuthHandler struct {
	invoked bool
	userID  string
}

type errorBody struct {
	Code      string           `json:"code"`
	Message   string           `json:"message"`
	RequestID string           `json:"request_id"`
	Details   *json.RawMessage `json:"details"`
}

func newJWTTestSetup(t *testing.T) (*gin.Engine, *auth.AuthService, *stubAuthHandler) {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	svc, err := auth.NewAuthService(nil, nil)
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID())
	group := router.Group("/v1/todos", JWT(svc))

	stub := &stubAuthHandler{}
	handler := func(c *gin.Context) {
		stub.invoked = true
		if uid, ok := c.Get("userID"); ok {
			stub.userID, _ = uid.(string)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	group.GET("", handler)
	group.POST("", handler)
	group.GET("/:id", handler)
	group.PUT("/:id", handler)
	group.DELETE("/:id", handler)

	return router, svc, stub
}

func doRequest(router *gin.Engine, method, path, authHeader string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	req.Header.Set("X-Request-ID", testRequestID)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

func assertUnauthorized(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 — body: %s", w.Code, w.Body.String())
	}
	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse 401 body: %v — %s", err, w.Body.String())
	}
	if body.Code != "unauthorized" {
		t.Errorf("code = %q, want %q", body.Code, "unauthorized")
	}
	if body.Message != testPassMessage {
		t.Errorf("message = %q, want %q", body.Message, testPassMessage)
	}
	if body.RequestID != testRequestID {
		t.Errorf("request_id = %q, want %q", body.RequestID, testRequestID)
	}
	if body.Details != nil {
		t.Errorf("details must be omitted, got %s", *body.Details)
	}
}

func signToken(t *testing.T, secret string, expiresAt time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
		Subject:   testUserID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestJWTValidTokenPasses(t *testing.T) {
	router, svc, stub := newJWTTestSetup(t)
	token, err := svc.GenerateAccessToken(testUserID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	w := doRequest(router, http.MethodGet, "/v1/todos", "Bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — body: %s", w.Code, w.Body.String())
	}
	if !stub.invoked {
		t.Fatal("downstream handler was not invoked")
	}
	if stub.userID != testUserID {
		t.Errorf("userID = %q, want %q", stub.userID, testUserID)
	}
}

func TestJWTAcceptsLowercaseScheme(t *testing.T) {
	router, svc, stub := newJWTTestSetup(t)
	token, err := svc.GenerateAccessToken(testUserID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	w := doRequest(router, http.MethodGet, "/v1/todos", "bearer "+token)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — body: %s", w.Code, w.Body.String())
	}
	if !stub.invoked {
		t.Fatal("downstream handler was not invoked")
	}
}

func TestJWTRejectsUnauthenticatedRequests(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	svc, err := auth.NewAuthService(nil, nil)
	if err != nil {
		t.Fatalf("NewAuthService: %v", err)
	}
	validToken, err := svc.GenerateAccessToken(testUserID)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}
	expired := signToken(t, testSecret, time.Now().Add(-time.Minute))
	badSignature := signToken(t, "a-different-secret", time.Now().Add(time.Minute))
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   testUserID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}

	cases := []struct {
		name       string
		authHeader string
	}{
		{"missing header", ""},
		{"raw token without scheme", validToken},
		{"wrong scheme", "Basic " + validToken},
		{"missing token part", "Bearer"},
		{"empty token after scheme", "Bearer "},
		{"malformed jwt", "Bearer not.a.jwt"},
		{"expired token", "Bearer " + expired},
		{"bad signature", "Bearer " + badSignature},
		{"unsupported signing method (alg none)", "Bearer " + unsigned},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, _, stub := newJWTTestSetup(t)
			w := doRequest(router, http.MethodGet, "/v1/todos", tc.authHeader)
			assertUnauthorized(t, w)
			if stub.invoked {
				t.Error("downstream handler was invoked despite authentication failure")
			}
		})
	}
}

func TestJWTProtectsAllTodoRoutes(t *testing.T) {
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/todos"},
		{http.MethodPost, "/v1/todos"},
		{http.MethodGet, "/v1/todos/123"},
		{http.MethodPut, "/v1/todos/123"},
		{http.MethodDelete, "/v1/todos/123"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			router, _, stub := newJWTTestSetup(t)
			w := doRequest(router, route.method, route.path, "")
			assertUnauthorized(t, w)
			if stub.invoked {
				t.Error("downstream handler was invoked despite missing token")
			}
		})
	}
}
