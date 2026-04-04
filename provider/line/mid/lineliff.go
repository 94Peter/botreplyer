package mid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/94peter/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type ctxKey string

const (
	keyUserId   ctxKey = "line.liff.userid"
	keyLocale   ctxKey = "line.liff.locale"
	keyUserName ctxKey = "line.liff.username"
	keyIsAdmin  ctxKey = "line.liff.isadmin"
	keyIsDemo   ctxKey = "is_demo"
)

const (
	sessionUserId   = "userId"
	sessionUserName = "userName"
	sessionLocale   = "locale"
	sessionIsAdmin  = "isAdmin"
)

// LineLiff is a middleware that extracts locale and userid from the request
// and saves them into the session.
func LineLiff() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		// Extract locale from path
		if lang := c.Param("lang"); lang != "" {
			oldLocale, _ := sess.Get(sessionLocale).(string)
			if oldLocale != lang {
				sess.Set(sessionLocale, lang)
			}
		}

		// Extract user info from user_token
		if token := c.Query("user_token"); token != "" {
			profile, err := getLineUserProfile(c.Request.Context(), token)
			if err != nil {
				log.Err(err)
				c.String(http.StatusUnauthorized, "Failed to get line user profile: %v", err)
				c.Abort()
				return
			}
			sess.Set(sessionUserId, profile.UserID)
			sess.Set(sessionUserName, profile.DisplayName)

			log.Infof("LineLiff: Logged in user %s (%s) from token", profile.DisplayName, profile.UserID)
		}

		userID, _ := sess.Get(sessionUserId).(string)
		locale, _ := sess.Get(sessionLocale).(string)
		userName, _ := sess.Get(sessionUserName).(string)

		// Set Gin Context Keys
		c.Set(string(keyUserId), userID)
		c.Set(string(keyLocale), locale)
		c.Set(string(keyUserName), userName)

		// Update Request Context for deep layers
		ctx := context.WithValue(c.Request.Context(), keyUserId, userID)
		ctx = context.WithValue(ctx, keyLocale, locale)
		ctx = context.WithValue(ctx, keyUserName, userName)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// SessionSaver ensuring that the session is saved at the end of the request.
func SessionSaver() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		sess := sessions.Default(c)
		if err := sess.Save(); err != nil {
			log.Errorf("SessionSaver: Failed to save session: %v", err)
		}
	}
}

type lineProfile struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl"`
	StatusMessage string `json:"statusMessage"`
}

func getLineUserProfile(ctx context.Context, accessToken string) (*lineProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.line.me/v2/profile", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("line API error: %s", string(body))
	}

	var profile lineProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func GetLineLiffUserId(c *gin.Context) string {
	return c.GetString(string(keyUserId))
}

func GetLineLiffLocale(c *gin.Context) string {
	return c.GetString(string(keyLocale))
}

func GetLineLiffUserName(c *gin.Context) string {
	return c.GetString(string(keyUserName))
}
