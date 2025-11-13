package mid

import (
	"context"
	"time"

	"github.com/94peter/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"github.com/94peter/botreplyer/follow"
)

const keyIsAdmin = "isAdmin"

func CheckAdmin(store follow.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		if val := sess.Get(keyIsAdmin); val != nil {
			isAdmin := val.(bool)
			setIsAdmin(c, isAdmin)
			c.Next()
			return
		}
		userId := c.GetString("line.liff.userid")
		if userId == "" {
			c.Set(keyIsAdmin, false)
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		f, err := store.Get(ctx, userId)
		cancel()
		if err != nil {
			log.Err(err)
			c.Set(keyIsAdmin, false)
			c.Next()
			return
		}
		setIsAdmin(c, f.IsAdmin())

		sess.Set(keyIsAdmin, f.IsAdmin())
		c.Next()
	}
}

func setIsAdmin(c *gin.Context, isAdmin bool) {
	c.Set(keyIsAdmin, isAdmin)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), keyIsAdmin, isAdmin))
}
