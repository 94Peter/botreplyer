package session

import (
	ginSessions "github.com/gin-contrib/sessions"
)

const KeyIsAdmin = "linebot-is-admin"

func IsKeyExist(session ginSessions.Session, key string) bool {
	return session.Get(key) != nil
}

func IsAdmin(session ginSessions.Session) bool {
	if session.Get(KeyIsAdmin) == nil {
		return false
	}
	val, ok := session.Get(KeyIsAdmin).(bool)
	if !ok {
		return false
	}
	return val
}
