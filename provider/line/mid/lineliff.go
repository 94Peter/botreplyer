package mid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/arwoosa/vulpes/log"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// LineLiff is a middleware that extracts locale and userid from the request
// and saves them into the session.
// The locale is extracted from the URL path (e.g., /en/some/path -> "en").
// The userid is extracted from the "userid" query parameter.
func LineLiff() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)

		// Extract locale from path, e.g., /en/training -> en
		var changed bool = false
		if lang := c.Param("lang"); lang != "" {
			sess.Set("locale", lang)
			changed = true
		}

		// Extract userid from query string, e.g., ?userid=1234
		if token := c.Query("user_token"); token != "" {
			profile, err := getLineUserProfile(token)
			if err != nil {
				log.Err(err)
				c.Writer.WriteHeader(http.StatusUnauthorized)
				c.Writer.Write([]byte("Failed to get line user profile: " + err.Error()))
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

		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), "line.liff.userid", userID))
		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), "line.liff.locale", userID))
		c.Request = c.Request.Clone(context.WithValue(c.Request.Context(), "line.liff.username", userID))

		// c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "line.liff.locale", locale))
		// c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "line.liff.username", userName))
		c.Set("line.liff.userid", userID)
		c.Set("line.liff.locale", locale)
		c.Set("line.liff.username", userName)
		c.Next()
	}
}

type lineProfile struct {
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	PictureURL    string `json:"pictureUrl"`
	StatusMessage string `json:"statusMessage"`
}

func getLineUserProfile(accessToken string) (*lineProfile, error) {
	req, err := http.NewRequest("GET", "https://api.line.me/v2/profile", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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
