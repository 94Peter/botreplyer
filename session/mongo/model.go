package mongo

import (
	"time"

	"github.com/94peter/vulpes/db/mgo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const sessionCollectionName = "session"

var sessionCollection = mgo.NewCollectDef(sessionCollectionName, func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
})

func init() {
	mgo.RegisterIndex(sessionCollection)
}

func NewSession() *Session {
	return &Session{
		Index: sessionCollection,
	}
}

type Session struct {
	mgo.Index `bson:"-"`
	ID        bson.ObjectID `bson:"_id"`
	UserID    string        `bson:"user_id"`
	Topic     string        `bson:"topic"`
	State     string        `bson:"state"`
	Context   bson.Raw      `bson:"context,omitempty"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

func (s *Session) GetTopic() string {
	return s.Topic
}
func (s *Session) SetTopic(topic string) error {
	s.Topic = topic
	return nil
}
func (s *Session) GetState() string {
	return s.State
}
func (s *Session) SetState(state string) error {
	s.State = state
	return nil
}

func (s *Session) GetId() any {
	return s.ID
}

func (s *Session) SetId(id any) {
	oid, ok := id.(bson.ObjectID)
	if !ok {
		return
	}
	s.ID = oid
}

func (p *Session) Validate() error {
	return nil
}
