package userrole

import (
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type UserRole struct {
	ID   shared.UserID
	Role shared.RoleID
}
