package mongo

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"time"

// 	"github.com/94peter/botreplyer/session"
// 	"github.com/94peter/vulpes/db/mgo"
// 	"go.mongodb.org/mongo-driver/v2/bson"
// 	"go.mongodb.org/mongo-driver/v2/mongo"
// )

// type mongoStore struct {
// }

// type MongoSession interface {
// 	session.Store
// }

// func NewSessionStore(ctx context.Context) (MongoSession, error) {
// 	if !mgo.IsConnected() {
// 		return nil, errors.New("mongo not connected")
// 	}
// 	exit, err := mgo.IsCollectionExist(ctx, sessionCollectionName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !exit {
// 		return nil, errors.New("collection not exist")
// 	}

// 	return &mongoStore{}, nil
// }

// func (store *mongoStore) NewSession(userID string) session.Session {
// 	result := NewSession()
// 	result.UserID = userID
// 	return result
// }
// func (store *mongoStore) UpsertSession(ctx context.Context, session session.Session) error {
// 	mysession, ok := session.(*Session)
// 	if !ok {
// 		return errors.New("invalid session type")
// 	}
// 	return store.upsert(ctx, mysession)
// }

// func (store *mongoStore) GetSession(ctx context.Context, userID string) (session.Session, error) {
// 	result := NewSession()
// 	err := mgo.FindOne(ctx, result, bson.M{"user_id": userID})
// 	if errors.Is(err, mongo.ErrNoDocuments) {
// 		return nil, nil
// 	}
// 	return result, err
// }

// func (store *mongoStore) Get(ctx context.Context, userID string) (*Session, error) {
// 	result := NewSession()
// 	err := mgo.FindOne(ctx, result, bson.M{"user_id": userID})
// 	if errors.Is(err, mongo.ErrNoDocuments) {
// 		return nil, nil
// 	}
// 	return result, err
// }

// func (store *mongoStore) upsert(ctx context.Context, session *Session) error {
// 	fmt.Println("upsert")
// 	session.UpdatedAt = time.Now()
// 	created := false
// 	if session.ID.IsZero() {
// 		session.ID = bson.NewObjectID()
// 		created = true
// 	}
// 	var err error
// 	if created {
// 		_, err = mgo.Save(ctx, session)
// 		return err
// 	} else {
// 		data := bson.M{
// 			"topic":      session.Topic,
// 			"updated_at": session.UpdatedAt,
// 			"state":      session.State,
// 		}
// 		if session.Context != nil {
// 			data["context"] = session.Context
// 		}
// 		_, err = mgo.UpdateById(ctx, session,
// 			bson.D{{Key: "$set", Value: data}})
// 	}

// 	return err
// }

// func (store *mongoStore) DeleteSession(ctx context.Context, userID string) error {
// 	_, err := mgo.DeleteOne(ctx, NewSession(), bson.D{{Key: "user_id", Value: userID}})
// 	return err
// }
