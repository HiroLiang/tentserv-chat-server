package email

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/jmoiron/sqlx"
)

type PostgresEmailRecorder struct {
	db *sqlx.DB
}

func NewPostgresEmailRecorder(db *sqlx.DB) *PostgresEmailRecorder {
	return &PostgresEmailRecorder{
		db: db,
	}
}

func (p PostgresEmailRecorder) RecordSending(ctx context.Context, email *shared.Email) {
	//TODO implement me
	panic("implement me")
}

func (p PostgresEmailRecorder) RecordFailed(ctx context.Context, email *shared.Email) {
	//TODO implement me
	panic("implement me")
}

func (p PostgresEmailRecorder) RecordSuccess(ctx context.Context, email *shared.Email, id string) {
	//TODO implement me
	panic("implement me")
}

var _ email.EmailRecorder = (*PostgresEmailRecorder)(nil)
