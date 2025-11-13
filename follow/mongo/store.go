package mongo

import (
	"context"
	"errors"

	"github.com/94peter/botreplyer/follow"
	"github.com/94peter/vulpes/db/mgo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type mongoStore struct{}

type MongoFollow interface {
	follow.Store
}

func NewStore(ctx context.Context) (MongoFollow, error) {
	if !mgo.IsConnected() {
		return nil, errors.New("mongo not connected")
	}
	exit, err := mgo.IsCollectionExist(ctx, followCollectionName)
	if err != nil {
		return nil, err
	}
	if !exit {
		return nil, errors.New("collection not exist")
	}

	return &mongoStore{}, nil
}

func (store *mongoStore) Delete(ctx context.Context, userID string) error {
	_, err := mgo.DeleteOne(ctx, NewFollow(), bson.D{{Key: "user_id", Value: userID}})
	return err
}

// 新增
func (store *mongoStore) Add(ctx context.Context, userID string, userName string, isAdmin bool) error {
	follow := NewFollow()
	follow.UserID = userID
	follow.IsSetAdmin = isAdmin
	follow.UserName = userName
	_, err := mgo.Save(ctx, follow)
	return err
}

// 取得
func (store *mongoStore) Get(ctx context.Context, userID string) (follow.Follow, error) {
	follow := NewFollow()
	err := mgo.FindOne(ctx, follow, bson.D{{Key: "user_id", Value: userID}})
	return follow, err
}

// 設定為管理員
func (store *mongoStore) Admin(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewFollow(),
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "is_admin", Value: true}}}})
	return err
}

// 取消管理員
func (store *mongoStore) UnAdmin(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewFollow(),
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "is_admin", Value: false}}}})
	return err
}
