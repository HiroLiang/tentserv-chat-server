package postgres

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/jmoiron/sqlx"
)

type txKey struct{}

type PostgresUnitOfWork struct {
	db *sqlx.DB
}

func NewPostgresUnitOfWork(db *sqlx.DB) *PostgresUnitOfWork {
	return &PostgresUnitOfWork{db: db}
}

func (u *PostgresUnitOfWork) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	tx, err := u.db.BeginTxx(ctx, nil)
	if err != nil {
		return ctx, nil, err
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	return txCtx, &PostgresTransaction{tx: tx}, nil
}

type PostgresTransaction struct {
	tx *sqlx.Tx
}

func (t PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

func (t PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

var _ transaction.UnitOfWork = (*PostgresUnitOfWork)(nil)

var _ transaction.Transaction = (*PostgresTransaction)(nil)
