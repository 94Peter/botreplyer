package mid

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/invopop/ctxi18n"
)

const (
	mockTrue = "true"
)

// MockAuth is a middleware that allows bypassing LINE login in Demo/Dev mode.
// It checks for identity overrides in Query, Cookie, or Header.
func MockAuth(isDemo bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(keyIsDemo), isDemo)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), keyIsDemo, isDemo))

		// Only enable if isDemo is true
		if !isDemo {
			c.Next()
			return
		}

		sess := sessions.Default(c)

		// 1. Force ensure locale exists to prevent layout.templ from jumping to liff-init.js
		const defaultLocale = "zh-tw"
		userLocale, _ := sess.Get(sessionLocale).(string)
		if userLocale == "" {
			userLocale = defaultLocale
			sess.Set(sessionLocale, userLocale)
			_ = sess.Save()
		}

		// Directly inject Request Context, which is most effective for layout.templ's i18n.GetLocale(ctx)
		newCtx, _ := ctxi18n.WithLocale(c.Request.Context(), userLocale)
		c.Request = c.Request.WithContext(newCtx)

		// Also set Gin Context so other middlewares can see it
		c.Set(string(keyLocale), userLocale)

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
			sess.Set(sessionUserId, mockUserID)
			sess.Set(sessionUserName, mockUserName)
			sess.Set(sessionIsAdmin, mockIsAdmin)
			_ = sess.Save()

			// Set Gin Context Keys
			setIdentity(c, mockUserID, mockUserName, mockIsAdmin)
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

// IsDemo returns true if the context identifies a Demo/Dev mode request.
func IsDemo(ctx context.Context) bool {
	val, _ := ctx.Value(keyIsDemo).(bool)
	return val
}

// GetLineUserName returns the current user name from context.
func GetLineUserName(ctx context.Context) string {
	val, _ := ctx.Value(keyUserName).(string)
	return val
}
