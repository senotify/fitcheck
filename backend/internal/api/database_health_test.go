package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"virtual-fitcheck/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestDatabaseHealthMiddleware_NilDatabase tests that the middleware returns 503 when database is nil
func TestDatabaseHealthMiddleware_NilDatabase(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Port:        "8080",
		StoragePath: t.TempDir(),
		DatabaseURL: "", // No database URL
	}

	// Create server without database
	server := &Server{
		config: cfg,
		db:     nil, // Explicitly nil database
	}

	// Create a test router with the middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(server.databaseHealthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert that we get 503 Service Unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "DATABASE_UNAVAILABLE")
	assert.Contains(t, w.Body.String(), "database service is currently unavailable")
}
