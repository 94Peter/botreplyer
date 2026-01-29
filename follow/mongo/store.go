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

func (*mongoStore) Delete(ctx context.Context, userID string) error {
	_, err := mgo.DeleteOne(ctx, NewFollow(), bson.D{{Key: "user_id", Value: userID}})
	return err
}

// 新增
func (*mongoStore) Add(ctx context.Context, userID string, userName string, isAdmin bool) error {
	follow := NewFollow()
	follow.UserID = userID
	follow.IsSetAdmin = isAdmin
	follow.UserName = userName
	_, err := mgo.Save(ctx, follow)
	return err
}

// 取得
func (*mongoStore) Get(ctx context.Context, userID string) (follow.Follow, error) {
	follow := NewFollow()
	err := mgo.FindOne(ctx, follow, bson.D{{Key: "user_id", Value: userID}})
	return follow, err
}

// 設定為管理員
func (*mongoStore) Admin(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewFollow(),
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "is_admin", Value: true}}}})
	return err
}

// 取消管理員
func (*mongoStore) UnAdmin(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewFollow(),
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "is_admin", Value: false}}}})
	return err
}
