package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create router with rate limit middleware
	router := gin.New()
	router.Use(rateLimitMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Make 105 requests from same IP
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 105; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345" // Same IP
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code == 200 {
			successCount++
		} else if w.Code == 429 {
			rateLimitedCount++
		}
	}

	t.Logf("Success: %d, Rate Limited: %d", successCount, rateLimitedCount)

	// Should allow ~100 requests and block ~5
	if successCount < 95 || successCount > 100 {
		t.Errorf("Expected ~100 successful requests, got %d", successCount)
	}
	if rateLimitedCount < 5 || rateLimitedCount > 10 {
		t.Errorf("Expected ~5 rate limited requests, got %d", rateLimitedCount)
	}
}
