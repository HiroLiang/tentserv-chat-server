package account

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/account"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/infrastructure/persistence/postgres"
	"github.com/jmoiron/sqlx"
)

type AccountRepo struct {
	postgres.BaseRepo
}

func NewAccountRepo(db *sqlx.DB) *AccountRepo {
	return &AccountRepo{
		BaseRepo: postgres.NewBaseRepo(db),
	}
}

func (r *AccountRepo) FindByID(ctx context.Context, id shared.AccountID) (*account.Account, error) {
	//TODO implement me
	panic("implement me")
}

func (r *AccountRepo) FindByAccountName(ctx context.Context, accountName string) (*account.Account, error) {
	//TODO implement me
	panic("implement me")
}

func (r *AccountRepo) FindByEmail(ctx context.Context, email shared.EmailAddress) (*account.Account, error) {
	//TODO implement me
	panic("implement me")
}

func (r *AccountRepo) Create(ctx context.Context, account *account.Account) (shared.AccountID, error) {
	//TODO implement me
	panic("implement me")
}

func (r *AccountRepo) Update(ctx context.Context, account *account.Account) error {
	//TODO implement me
	panic("implement me")
}

var _ account.Repository = (*AccountRepo)(nil)

var AccountTable = postgres.Table{
	Name: "goat.public.accounts",
	Columns: []string{
		"id",
		"public_id",
		"email",
		"account",
		"password",
		"status",
		"user_limit",
		"created_at",
		"updated_at",
	},
}
