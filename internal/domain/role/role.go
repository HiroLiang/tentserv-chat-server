package role

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Role struct {
	ID          shared.RoleID
	Code        Code
	Name        string
	Description string
	Creator     shared.AccountID
	CreateAt    time.Time
	UpdatedAt   time.Time
}
