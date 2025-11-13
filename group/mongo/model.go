package mongo

import (
	"time"

	"github.com/arwoosa/vulpes/db/mgo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const groupCollectionName = "group"

var groupCollection = mgo.NewCollectDef(groupCollectionName, func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "group_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
})

func init() {
	mgo.RegisterIndex(groupCollection)
}

func NewGroup() *Group {
	return &Group{
		Index:     groupCollection,
		ID:        bson.NewObjectID(),
		CreatedAt: time.Now(),
	}
}

type Group struct {
	mgo.Index `bson:"-"`
	ID        bson.ObjectID `bson:"_id"`
	GroupID   string        `bson:"group_id"`
	Name      string        `bson:"name"`
	Disable   bool          `bson:"disable"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (s *Group) GetId() any {
	return s.ID
}

func (s *Group) SetId(id any) {
	oid, ok := id.(bson.ObjectID)
	if !ok {
		return
	}
	s.ID = oid
}

func (p *Group) Validate() error {
	return nil
}

func (p *Group) IsDisabled() bool {
	return p.Disable
}
