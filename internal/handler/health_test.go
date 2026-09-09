package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"todo-list-api/internal/db"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func doHealthRequest(t *testing.T, db *sql.DB, method, path string) (int, map[string]string) {
	t.Helper()
	router := gin.New()
	RegisterHealthRoutes(router, db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return w.Code, resp
}

func TestHealthLiveness(t *testing.T) {
	testDB := db.SetupTestDB(t)
	code, resp := doHealthRequest(t, testDB, "GET", "/health")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status \"ok\", got %q", resp["status"])
	}
}

func TestHealthReadinessOK(t *testing.T) {
	testDB := db.SetupTestDB(t)
	code, resp := doHealthRequest(t, testDB, "GET", "/health/ready")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status \"ok\", got %q", resp["status"])
	}
}

func TestHealthReadinessUnavailable(t *testing.T) {
	testDB := db.SetupTestDB(t)
	_ = testDB.Close() // Close DB to simulate failure
	code, resp := doHealthRequest(t, testDB, "GET", "/health/ready")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", code)
	}
	if resp["status"] != "unavailable" {
		t.Fatalf("expected status \"unavailable\", got %q", resp["status"])
	}
}
