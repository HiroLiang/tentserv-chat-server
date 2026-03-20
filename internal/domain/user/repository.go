package user

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	Create(ctx context.Context, user *User) (shared.UserID, error)
	FindByID(ctx context.Context, id shared.UserID) (*User, error)
	FindByAccountID(ctx context.Context, accountID shared.AccountID) (*[]User, error)
	Update(ctx context.Context, user *User) error
	SearchByName(ctx context.Context, keyword string) ([]*UserSearchResult, error)
	FindByAccountName(ctx context.Context, accountName string) ([]*UserSearchResult, error)
	FindByPublicID(ctx context.Context, publicID string) ([]*UserSearchResult, error)
}
