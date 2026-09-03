package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDGenerated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	headerID := w.Header().Get("X-Request-ID")
	if headerID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	if len(headerID) != 36 {
		t.Errorf("expected UUID format (36 chars), got %d chars: %s", len(headerID), headerID)
	}
}

func TestRequestIDRespected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	router.ServeHTTP(w, req)

	headerID := w.Header().Get("X-Request-ID")
	if headerID != "custom-id-123" {
		t.Errorf("expected custom-id-123, got %s", headerID)
	}
}

func TestRequestIDInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var capturedID string

	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		val, exists := c.Get("requestID")
		if !exists {
			t.Error("requestID not found in context")
			return
		}
		capturedID = val.(string)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	headerID := w.Header().Get("X-Request-ID")
	if capturedID != headerID {
		t.Errorf("context requestID=%s, header=%s", capturedID, headerID)
	}
}
