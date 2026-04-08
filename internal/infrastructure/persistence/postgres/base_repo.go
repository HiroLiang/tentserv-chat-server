package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// [EN] BaseRepo provides GetDB(ctx) which returns the active transaction stored in context (if any),
//      otherwise returns the raw DB connection. This enables transparent transaction support:
//      use cases pass a context with a tx, and all repos inside the same tx use it automatically.
// [中] BaseRepo 提供 GetDB(ctx)，若 context 中有活躍的交易則返回它，否則返回原始 DB 連線。
//      這使 Use Case 可透明地傳遞 context 中的 tx，同一交易中的所有 Repo 自動使用它。
// [日] BaseRepo は GetDB(ctx) を提供し、コンテキストにアクティブなトランザクションがあればそれを、
//      なければ生の DB 接続を返す。これにより、ユースケースが context に tx を渡すと
//      同一トランザクション内の全 Repo が自動的にそれを使用できる透過的なトランザクション機能を実現する。
type BaseRepo struct {
	db *sqlx.DB
}

func NewBaseRepo(db *sqlx.DB) BaseRepo {
	return BaseRepo{db: db}
}

func (r *BaseRepo) GetDB(ctx context.Context) sqlx.ExtContext {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return r.db
}
