package userrole

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type UserRole struct {
	ID   shared.UserID
	Role shared.RoleID
}
