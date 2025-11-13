package group

import "context"

type Group interface {
	IsDisabled() bool
}

type Store interface {
	// 新增
	Add(ctx context.Context, groupID string, active bool) error
	// 取得
	Get(ctx context.Context, groupID string) (Group, error)
	// 刪除
	Delete(ctx context.Context, groupID string) error
	// 啟用Group
	Active(ctx context.Context, groupID string) error
	// 停用Group
	Inactive(ctx context.Context, groupID string) error
}
