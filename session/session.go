package session

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"

	ginSessions "github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DefaultKey  = "botreplyer/sessions"
	errorFormat = "[sessions] ERROR!"
)

func UserIdToObjectID(userId string) string {
	sum := sha256.Sum256([]byte(userId))
	// 取前 12 bytes（24 hex 字元）
	hexStr := hex.EncodeToString(sum[:12])
	oid, _ := bson.ObjectIDFromHex(hexStr)
	return oid.Hex()
}

func NewSessionFromSession(c *gin.Context, name string, s *sessions.Session, store sessions.Store) ginSessions.Session {
	return &session{
		name:    name,
		request: c.Request,
		store:   store,
		written: false,
		session: s,
		writer:  c.Writer,
	}
}

type session struct {
	store   sessions.Store
	writer  http.ResponseWriter
	request *http.Request
	session *sessions.Session
	name    string
	written bool
}

func (s *session) ID() string {
	return s.Session().ID
}

func (s *session) Get(key any) any {
	return s.Session().Values[key]
}

func (s *session) Set(key any, val any) {
	s.Session().Values[key] = val
	s.written = true
}

func (s *session) Delete(key any) {
	delete(s.Session().Values, key)
	s.written = true
}

func (s *session) Clear() {
	for key := range s.Session().Values {
		s.Delete(key)
	}
}

func (s *session) AddFlash(value any, vars ...string) {
	s.Session().AddFlash(value, vars...)
	s.written = true
}

func (s *session) Flashes(vars ...string) []any {
	s.written = true
	return s.Session().Flashes(vars...)
}

func (s *session) Options(options ginSessions.Options) {
	s.written = true
	s.Session().Options = options.ToGorillaOptions()
}

func (s *session) Save() error {
	if s.Written() {
		e := s.Session().Save(s.request, s.writer)
		if e == nil {
			s.written = false
		}
		return e
	}
	return nil
}

func (s *session) Session() *sessions.Session {
	if s.session == nil {
		var err error
		s.session, err = s.store.Get(s.request, s.name)
		if err != nil {
			slog.Error(errorFormat,
				"err", err,
			)
		}
	}
	return s.session
}

func (s *session) Written() bool {
	return s.written
}
