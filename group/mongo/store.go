package mongo

import (
	"context"
	"errors"

	"github.com/94peter/botreplyer/group"

	"github.com/94peter/vulpes/db/mgo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type mongoStore struct{}

func NewStore(ctx context.Context) (group.Store, error) {
	if !mgo.IsConnected() {
		return nil, errors.New("mongo not connected")
	}
	exit, err := mgo.IsCollectionExist(ctx, groupCollectionName)
	if err != nil {
		return nil, err
	}
	if !exit {
		return nil, errors.New("collection not exist")
	}
	return &mongoStore{}, nil
}

func (*mongoStore) Delete(ctx context.Context, groupID string) error {
	_, err := mgo.DeleteOne(ctx, NewGroup(), bson.D{{Key: "group_id", Value: groupID}})
	return err
}

// 新增
func (*mongoStore) Add(ctx context.Context, groupID string, active bool) error {
	obj := NewGroup()
	obj.GroupID = groupID
	obj.Disable = !active
	_, err := mgo.Save(ctx, obj)
	return err
}

// 取得
func (*mongoStore) Get(ctx context.Context, userID string) (group.Group, error) {
	obj := NewGroup()
	err := mgo.FindOne(ctx, obj, bson.D{{Key: "group_id", Value: userID}})
	return obj, err
}

// 設定為管理員
func (*mongoStore) Active(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewGroup(),
		bson.D{{Key: "group_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "disable", Value: false}}}})
	return err
}

// 取消管理員
func (*mongoStore) Inactive(ctx context.Context, userID string) error {
	_, err := mgo.UpdateOne(
		ctx, NewGroup(),
		bson.D{{Key: "group_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "disable", Value: true}}}})
	return err
}
