package follow

import "context"

type Follow interface {
	IsAdmin() bool
}

type Store interface {
	// 新增
	Add(ctx context.Context, userID string, userName string, isAdmin bool) error
	// 取得
	Get(ctx context.Context, userID string) (Follow, error)
	// 刪除
	Delete(ctx context.Context, userID string) error
	// 設定為管理員
	Admin(ctx context.Context, userID string) error
	// 取消管理員
	UnAdmin(ctx context.Context, userID string) error
}
