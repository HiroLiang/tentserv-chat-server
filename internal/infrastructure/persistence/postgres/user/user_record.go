package user

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type UserRecord struct {
	ID         user.ID     `db:"id"`
	Name       string      `db:"name"`
	Email      user.Email  `db:"email"`
	Password   string      `db:"password"`
	UserStatus user.Status `db:"user_status"`
	UserIP     string      `db:"user_ip"`
	AvatarName *string     `db:"avatar_name"`
	CreatedAt  time.Time   `db:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at"`
}
