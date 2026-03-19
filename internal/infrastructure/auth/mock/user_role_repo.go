package mock

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
)

type UserRoleRepo struct{}

func MockUserRoleRepo() *UserRoleRepo {
	return &UserRoleRepo{}
}

var _ userrole.Repository = (*UserRoleRepo)(nil)

func (u UserRoleRepo) FindRolesByUser(ctx context.Context, userID shared.UserID) ([]*role.Role, error) {
	panic("implement me")
}

func (u UserRoleRepo) Exists(ctx context.Context, userID shared.UserID, role role.Code) bool {
	panic("implement me")
}

func (u UserRoleRepo) Assign(ctx context.Context, userID shared.UserID, role role.Code) error {
	return nil
}

func (u UserRoleRepo) Revoke(ctx context.Context, userID shared.UserID, role role.Code) error {
	panic("implement me")
}
