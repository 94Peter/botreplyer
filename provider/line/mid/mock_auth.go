package mid

import (
	"context"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/invopop/ctxi18n"
)

const (
	mockTrue       = "true"
	defaultID      = "coach_demo"
	defaultName    = "Sean (教練主理人)"
	defaultIsAdmin = true
)

// MockAuth is a middleware that allows bypassing LINE login in Demo/Dev mode.
func MockAuth(isDemo bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Always set the isDemo flag
		c.Set(string(keyIsDemo), isDemo)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), keyIsDemo, isDemo))

		if !isDemo {
			c.Next()
			return
		}

		sess := sessions.Default(c)

		// 1. Manage Locale (I18n)
		ensureLocale(c, sess)

		// 2. Resolve Mock Identity
		userID, userName, isAdmin := resolveIdentity(c, sess)

		// 3. Persist and Inject Identity
		sess.Set(sessionUserId, userID)
		sess.Set(sessionUserName, userName)
		sess.Set(sessionIsAdmin, isAdmin)
		_ = sess.Save()

		setIdentity(c, userID, userName, isAdmin)

		c.Next()
	}
}

// ensureLocale ensures a locale exists in the session and injects it into both Gin and Request contexts.
func ensureLocale(c *gin.Context, sess sessions.Session) {
	const defaultLocale = "zh-tw"
	userLocale, _ := sess.Get(sessionLocale).(string)
	if userLocale == "" {
		userLocale = defaultLocale
		sess.Set(sessionLocale, userLocale)
		_ = sess.Save()
	}

	// Directly inject Request Context for layout.templ's i18n helper
	newCtx, _ := ctxi18n.WithLocale(c.Request.Context(), userLocale)
	c.Request = c.Request.WithContext(newCtx)

	// Set Gin Context for other middlewares
	c.Set(string(keyLocale), userLocale)
}

// resolveIdentity determines the current user identity based on a priority queue:
// Query/Header > Session > Default Fallback
func resolveIdentity(c *gin.Context, sess sessions.Session) (id, name string, isAdmin bool) {
	// A. Check Request-level overrides (Query or Headers)
	id = getFirst(c.Query("mock_user_id"), c.GetHeader("X-Mock-User-ID"))
	name = getFirst(c.Query("mock_user_name"), c.GetHeader("X-Mock-User-Name"))

	// B. If no request override, try persistent Session
	if id == "" {
		if sID, ok := sess.Get(sessionUserId).(string); ok {
			id = sID
			if sName, ok := sess.Get(sessionUserName).(string); ok {
				name = sName
			}
		}
	}

	// C. Resolve Admin status if an ID exists
	if id != "" {
		isAdmin = c.Query("mock_is_admin") == mockTrue || c.GetHeader("X-Mock-Is-Admin") == mockTrue
		if !isAdmin {
			if sAdmin, ok := sess.Get(sessionIsAdmin).(bool); ok {
				isAdmin = sAdmin
			}
		}
		return
	}

	// D. Final Fallback to hardcoded defaults
	return defaultID, defaultName, defaultIsAdmin
}

func getFirst(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
