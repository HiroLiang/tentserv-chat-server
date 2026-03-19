package userrole

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindRolesByUser(ctx context.Context, userID shared.UserID) ([]*role.Role, error)
	Exists(ctx context.Context, userID shared.UserID, role role.Code) bool
	Assign(ctx context.Context, userID shared.UserID, role role.Code) error
	Revoke(ctx context.Context, userID shared.UserID, role role.Code) error
}
