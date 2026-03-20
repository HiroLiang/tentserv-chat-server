package mock

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type UserRepo struct{}

func MockUserRepo() *UserRepo {
	return &UserRepo{}
}

var _ user.Repository = (*UserRepo)(nil)

func (u *UserRepo) Create(ctx context.Context, usr *user.User) (shared.UserID, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) FindByID(ctx context.Context, id shared.UserID) (*user.User, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) FindByAccountID(ctx context.Context, accountID shared.AccountID) (*[]user.User, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) Update(ctx context.Context, usr *user.User) error {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) SearchByName(ctx context.Context, keyword string) ([]*user.UserSearchResult, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) FindByAccountName(ctx context.Context, accountName string) ([]*user.UserSearchResult, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) FindByPublicID(ctx context.Context, publicID string) ([]*user.UserSearchResult, error) {
	//TODO implement me
	panic("implement me")
}
