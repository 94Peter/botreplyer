package mongo

import (
	"time"

	"github.com/arwoosa/vulpes/db/mgo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const followCollectionName = "follow"

var followCollection = mgo.NewCollectDef(followCollectionName, func() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
})

func init() {
	mgo.RegisterIndex(followCollection)
}

func NewFollow() *Follow {
	return &Follow{
		Index:     followCollection,
		ID:        bson.NewObjectID(),
		CreatedAt: time.Now(),
	}
}

type Follow struct {
	mgo.Index  `bson:"-"`
	ID         bson.ObjectID `bson:"_id"`
	UserID     string        `bson:"user_id"`
	UserName   string        `bson:"user_name"`
	IsSetAdmin bool          `bson:"is_admin"`
	CreatedAt  time.Time     `bson:"created_at"`
}

func (s *Follow) GetId() any {
	return s.ID
}

func (s *Follow) SetId(id any) {
	oid, ok := id.(bson.ObjectID)
	if !ok {
		return
	}
	s.ID = oid
}

func (p *Follow) Validate() error {
	return nil
}

func (p *Follow) IsAdmin() bool {
	return p.IsSetAdmin
}
