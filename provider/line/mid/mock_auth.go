package mid

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	mockTrue             = "true"
	isDemoKey contextKey = "is_demo"
)

// MockAuth is a middleware that allows bypassing LINE login in Demo/Dev mode.
// It checks for identity overrides in Query, Cookie, or Header.
func MockAuth(isDemo bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("is_demo", isDemo)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), isDemoKey, isDemo))
		// Only enable if isDemo is true
		if !isDemo {
			c.Next()
			return
		}

		sess := sessions.Default(c)

		// 1. Check for mock_user_id
		mockUserID := c.Query("mock_user_id")
		if mockUserID == "" {
			if cookie, err := c.Cookie("mock_user_id"); err == nil {
				mockUserID = cookie
			}
		}
		if mockUserID == "" {
			mockUserID = c.GetHeader("X-Mock-User-ID")
		}

		// 2. Check for mock_user_name
		mockUserName := c.Query("mock_user_name")
		if mockUserName == "" {
			mockUserName = c.GetHeader("X-Mock-User-Name")
		}
		if mockUserName == "" {
			mockUserName = "Demo User"
		}

		// 3. Check for mock_is_admin
		mockIsAdmin := c.Query("mock_is_admin") == mockTrue || c.GetHeader("X-Mock-Is-Admin") == mockTrue

		if mockUserID != "" {
			// Save to session so it persists for subsequent requests
			sess.Set("userId", mockUserID)
			sess.Set("userName", mockUserName)
			sess.Set(string(keyIsAdmin), mockIsAdmin)
			_ = sess.Save()

			// Set Gin Context Keys
			setIdentity(c, mockUserID, mockUserName, mockIsAdmin)

			// If we explicitly provided a mock_user_id, we might want to skip LineLiff logic
			// but we can also just let LineLiff run and it will see the session is already set?
			// Actually LineLiff only sets it if user_token is present or session is empty.
			// Let's see LineLiff logic again.
		}

		c.Next()
	}
}

func setIdentity(c *gin.Context, userID, userName string, isAdmin bool) {
	c.Set(string(keyUserId), userID)
	c.Set(string(keyUserName), userName)
	c.Set(string(keyIsAdmin), isAdmin)

	// Update Request Context
	ctx := context.WithValue(c.Request.Context(), keyUserId, userID)
	ctx = context.WithValue(ctx, keyUserName, userName)
	ctx = context.WithValue(ctx, keyIsAdmin, isAdmin)
	c.Request = c.Request.WithContext(ctx)
}
