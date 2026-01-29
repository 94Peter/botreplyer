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

type ctxLineLiffKey string

const (
	CtxLineLiffUserId   ctxLineLiffKey = "line.liff.userid"
	CtxLineLiffLocale   ctxLineLiffKey = "line.liff.locale"
	CtxLineLiffUserName ctxLineLiffKey = "line.liff.username"
)

// LineLiff is a middleware that extracts locale and userid from the request
// and saves them into the session.
// The locale is extracted from the URL path (e.g., /en/some/path -> "en").
// The userid is extracted from the "userid" query parameter.
func LineLiff() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		// Extract locale from path, e.g., /en/training -> en
		var changed bool
		if lang := c.Param("lang"); lang != "" {
			sess.Set("locale", lang)
			changed = true
		}

		// Extract userid from query string, e.g., ?userid=1234
		if token := c.Query("user_token"); token != "" {
			profile, err := getLineUserProfile(c.Request.Context(), token)
			if err != nil {
				log.Err(err)
				c.String(http.StatusUnauthorized, "Failed to get line user profile: %v", err)
				c.Abort()
				return
			}
			sess.Set("userId", profile.UserID)
			sess.Set("userName", profile.DisplayName)
			changed = true
		}

		if changed {
			if err := sess.Save(); err != nil {
				// Depending on the desired behavior, you might want to log this error
				// or handle it more gracefully. For now, we'll let the request proceed.
				log.Err(err)
			}
		}

		userID := sess.Get("userId")
		locale := sess.Get("locale")
		userName := sess.Get("userName")

		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), CtxLineLiffUserId, userID))
		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), CtxLineLiffLocale, userID))
		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), CtxLineLiffUserName, userID))

		c.Set(CtxLineLiffUserId, userID)
		c.Set(CtxLineLiffLocale, locale)
		c.Set(CtxLineLiffUserName, userName)
		c.Next()
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
		if err := resp.Body.Close(); err != nil {
			log.Err(err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("line API error: %s", string(body))
	}

	var profile lineProfile
	err = json.NewDecoder(resp.Body).Decode(&profile)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
