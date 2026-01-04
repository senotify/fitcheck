package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "fitcheck_session"
	
	// SessionCookieMaxAge is the max age of the session cookie in seconds (30 days)
	SessionCookieMaxAge = 30 * 24 * 60 * 60
)

// sessionMiddleware creates or retrieves a session ID for each request
// Requirements: 1.1, 2.2
func sessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get existing session ID from cookie
		sessionID, err := c.Cookie(SessionCookieName)
		
		// If no session exists or cookie is invalid, create a new session
		if err != nil || sessionID == "" {
			// Generate new UUID v4 for session identifier
			sessionID = uuid.New().String()
			
			// Set secure HTTP-only cookie with 30-day expiration
			c.SetCookie(
				SessionCookieName,     // name
				sessionID,             // value
				SessionCookieMaxAge,   // maxAge (30 days in seconds)
				"/",                   // path
				"",                    // domain (empty = current domain)
				false,                 // secure (set to true in production with HTTPS)
				true,                  // httpOnly (prevents JavaScript access)
			)
		}
		
		// Store session ID in context for handlers to access
		c.Set("sessionID", sessionID)
		
		c.Next()
	}
}

// GetSessionID retrieves the session ID from the request context
func GetSessionID(c *gin.Context) string {
	sessionID, exists := c.Get("sessionID")
	if !exists {
		return ""
	}
	
	if id, ok := sessionID.(string); ok {
		return id
	}
	
	return ""
}
