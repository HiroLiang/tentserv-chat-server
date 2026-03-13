package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
)

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
