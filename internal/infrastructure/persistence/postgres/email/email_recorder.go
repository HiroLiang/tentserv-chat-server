package email

import (
	"context"
	"strings"

	appEmail "github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/jmoiron/sqlx"
)

var Table = postgres.Table{
	Name: "goat.public.email_logs",
	Columns: []string{
		"id",
		"sender",
		"recipients",
		"subject",
		"status",
		"external_id",
		"created_at",
	},
}

type PostgresEmailRecorder struct {
	db *sqlx.DB
}

func NewPostgresEmailRecorder(db *sqlx.DB) *PostgresEmailRecorder {
	return &PostgresEmailRecorder{db: db}
}

var _ appEmail.EmailRecorder = (*PostgresEmailRecorder)(nil)

func (p *PostgresEmailRecorder) RecordSending(ctx context.Context, mail *shared.Email) {
	query, args, err := Table.Insert().
		Columns("sender", "recipients", "subject", "status").
		Values(mail.Sender.String(), joinRecipients(mail.Recipients), mail.Subject, "sending").
		ToSql()
	if err != nil {
		logger.Log.Error(err.Error())
		return
	}
	if err := postgres.Exec(ctx, p.db, query, args...); err != nil {
		logger.Log.Error(err.Error())
	}
}

func (p *PostgresEmailRecorder) RecordFailed(ctx context.Context, mail *shared.Email) {
	query, args, err := Table.Insert().
		Columns("sender", "recipients", "subject", "status").
		Values(mail.Sender.String(), joinRecipients(mail.Recipients), mail.Subject, "failed").
		ToSql()
	if err != nil {
		logger.Log.Error(err.Error())
		return
	}
	if err := postgres.Exec(ctx, p.db, query, args...); err != nil {
		logger.Log.Error(err.Error())
	}
}

func (p *PostgresEmailRecorder) RecordSuccess(ctx context.Context, mail *shared.Email, id string) {
	query, args, err := Table.Insert().
		Columns("sender", "recipients", "subject", "status", "external_id").
		Values(mail.Sender.String(), joinRecipients(mail.Recipients), mail.Subject, "sent", id).
		ToSql()
	if err != nil {
		logger.Log.Error(err.Error())
		return
	}
	if err := postgres.Exec(ctx, p.db, query, args...); err != nil {
		logger.Log.Error(err.Error())
	}
}

func joinRecipients(recipients []shared.EmailAddress) string {
	return strings.Join(shared.ToStringSlice(recipients), ",")
}
