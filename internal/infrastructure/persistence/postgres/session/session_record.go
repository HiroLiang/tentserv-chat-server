package session

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
)

type SessionRecord struct {
	ID               int64      `db:"id"`
	AccountID        int64      `db:"account_id"`
	UserID           int64      `db:"user_id"`
	DeviceID         string     `db:"device_id"`
	RefreshTokenHash string     `db:"refresh_token_hash"`
	ExpiresAt        time.Time  `db:"expires_at"`
	Revoked          bool       `db:"revoked"`
	CreatedAt        time.Time  `db:"created_at"`
	LastUsedAt       *time.Time `db:"last_used_at"`
}

var Table = postgres.Table{
	Name: "public.account_sessions",
	Columns: []string{
		"id",
		"account_id",
		"user_id",
		"device_id",
		"refresh_token_hash",
		"expires_at",
		"revoked",
		"created_at",
		"last_used_at",
	},
}
