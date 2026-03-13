package account

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, id shared.AccountID) (*Account, error)
	FindByAccountName(ctx context.Context, accountName string) (*Account, error)
	FindByEmail(ctx context.Context, email shared.EmailAddress) (*Account, error)
	Create(ctx context.Context, account *Account) (shared.AccountID, error)
	Update(ctx context.Context, account *Account) error
}
