package mid

import (
	"context"
	"time"

	"github.com/94peter/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/94peter/botreplyer/follow"
)

func CheckAdmin(store follow.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		if val := sess.Get(sessionIsAdmin); val != nil {
			isAdmin, ok := val.(bool)
			if !ok {
				isAdmin = false
			}
			setIsAdmin(c, isAdmin)
			c.Next()
			return
		}

		userId := c.GetString(string(keyUserId))
		if userId == "" {
			setIsAdmin(c, false)
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		f, err := store.Get(ctx, userId)
		if err != nil {
			log.Err(err)
			setIsAdmin(c, false)
			c.Next()
			return
		}

		isAdmin := f.IsAdmin()
		setIsAdmin(c, isAdmin)

		sess.Set(sessionIsAdmin, isAdmin)
		c.Next()
	}
}

func setIsAdmin(c *gin.Context, isAdmin bool) {
	c.Set(string(keyIsAdmin), isAdmin)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), keyIsAdmin, isAdmin))
}

func IsAdmin(ctx context.Context) bool {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return ginCtx.GetBool(string(keyIsAdmin))
	}
	val, _ := ctx.Value(keyIsAdmin).(bool)
	return val
}
