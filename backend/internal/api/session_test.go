package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"virtual-fitcheck/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestSessionMiddleware_CreatesNewSession(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Make a request without a session cookie
	req := httptest.NewRequest("GET", "/api/preview/test-file", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Check that a session cookie was set
	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies, "Expected session cookie to be set")

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	assert.NotNil(t, sessionCookie, "Expected fitcheck_session cookie")
	assert.NotEmpty(t, sessionCookie.Value, "Session ID should not be empty")
	assert.Equal(t, SessionCookieMaxAge, sessionCookie.MaxAge, "Cookie max age should be 30 days")
	assert.True(t, sessionCookie.HttpOnly, "Cookie should be HTTP-only")
	assert.Equal(t, "/", sessionCookie.Path, "Cookie path should be /")
}

func TestSessionMiddleware_ReusesExistingSession(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// First request - creates session
	req1 := httptest.NewRequest("GET", "/api/preview/test-file", nil)
	w1 := httptest.NewRecorder()
	server.router.ServeHTTP(w1, req1)

	// Get the session cookie from first request
	cookies1 := w1.Result().Cookies()
	var sessionCookie1 *http.Cookie
	for _, cookie := range cookies1 {
		if cookie.Name == SessionCookieName {
			sessionCookie1 = cookie
			break
		}
	}
	assert.NotNil(t, sessionCookie1)
	sessionID1 := sessionCookie1.Value

	// Second request - with existing session cookie
	req2 := httptest.NewRequest("GET", "/api/preview/test-file", nil)
	req2.AddCookie(sessionCookie1)
	w2 := httptest.NewRecorder()
	server.router.ServeHTTP(w2, req2)

	// Get the session cookie from second request
	cookies2 := w2.Result().Cookies()
	
	// The session ID should be the same (reused)
	// Note: Gin may or may not set the cookie again if it already exists
	// So we check if either no cookie is set (reused) or the same cookie is set
	if len(cookies2) > 0 {
		var sessionCookie2 *http.Cookie
		for _, cookie := range cookies2 {
			if cookie.Name == SessionCookieName {
				sessionCookie2 = cookie
				break
			}
		}
		if sessionCookie2 != nil {
			assert.Equal(t, sessionID1, sessionCookie2.Value, "Session ID should be reused")
		}
	}
}

func TestGetSessionID(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:          "8080",
		StoragePath:   t.TempDir(),
		CleanupDelay:  60,
		MaxUploadSize: 10 * 1024 * 1024,
	}
	server := NewServer(cfg)

	// Make a request and verify session ID is accessible in context
	req := httptest.NewRequest("GET", "/api/preview/test-file", nil)
	w := httptest.NewRecorder()

	// We need to test this through the actual router to ensure middleware runs
	server.router.ServeHTTP(w, req)

	// The session middleware should have run and set a session cookie
	cookies := w.Result().Cookies()
	assert.NotEmpty(t, cookies, "Expected session cookie to be set")
}
