package session

import (
	ginSessions "github.com/gin-contrib/sessions"
)

const KeyIsAdmin = "linebot-is-admin"

func IsKeyExist(session ginSessions.Session, key string) bool {
	if session.Get(key) == nil {
		return false
	}
	return true
}

func IsAdmin(session ginSessions.Session) bool {
	if session.Get(KeyIsAdmin) == nil {
		return false
	}
	return session.Get(KeyIsAdmin).(bool)
}
