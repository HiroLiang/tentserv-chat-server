package user

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type UserRecord struct {
	ID         shared.UserID    `db:"id"`
	AccountID  shared.AccountID `db:"account_id"`
	Name       string           `db:"name"`
	AvatarName *string          `db:"avatar"`
	CreatedAt  time.Time        `db:"created_at"`
	UpdatedAt  time.Time        `db:"updated_at"`
}
