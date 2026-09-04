package handler

import (
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

func TestHealthLiveness(t *testing.T) {
	testDB := db.SetupTestDB(t)

	router := gin.New()
	RegisterHealthRoutes(router, testDB)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status \"ok\", got %q", resp["status"])
	}
}

func TestHealthReadinessOK(t *testing.T) {
	testDB := db.SetupTestDB(t)

	router := gin.New()
	RegisterHealthRoutes(router, testDB)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status \"ok\", got %q", resp["status"])
	}
}

func TestHealthReadinessUnavailable(t *testing.T) {
	testDB := db.SetupTestDB(t)
	// Close DB to simulate failure
	_ = testDB.Close()

	router := gin.New()
	RegisterHealthRoutes(router, testDB)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["status"] != "unavailable" {
		t.Fatalf("expected status \"unavailable\", got %q", resp["status"])
	}
}
