package mock

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type UserRepo struct{}

func MockUserRepo() *UserRepo {
	return &UserRepo{}
}

var _ user.Repository = (*UserRepo)(nil)

func (u *UserRepo) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) CreateWithRole(ctx context.Context, user *user.User, roleType string) error {
	//TODO implement me
	panic("implement me")
}

func (u *UserRepo) Update(ctx context.Context, user *user.User) error {
	//TODO implement me
	panic("implement me")
}
