package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/jmoiron/sqlx"
)

func ScanOne[T any](ctx context.Context, db sqlx.ExtContext, query string, args ...any) (*T, error) {
	var rec T
	if err := sqlx.GetContext(ctx, db, &rec, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan one: %w", err)
	}
	return &rec, nil
}

func ScanAll[T any](ctx context.Context, db sqlx.ExtContext, query string, args ...any) ([]T, error) {
	var list []T
	if err := sqlx.SelectContext(ctx, db, &list, query, args...); err != nil {
		return nil, fmt.Errorf("scan all: %w", err)
	}
	return list, nil
}

func Exec(ctx context.Context, db sqlx.ExtContext, query string, args ...any) error {
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("exec query: %w", err)
	}
	return nil
}

func Exists(ctx context.Context, db sqlx.ExtContext, query string, args ...any) bool {
	var one int
	if err := db.QueryRowxContext(ctx, query, args...).Scan(&one); err != nil {
		logger.Log.Error(err.Error())
		return false
	}
	return one > 0
}
