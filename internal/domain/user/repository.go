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
	SearchByName(ctx context.Context, keyword string, limit, offset int) ([]*UserSearchResult, error)
	FindByAccountName(ctx context.Context, accountName string, limit, offset int) ([]*UserSearchResult, error)
	FindByPublicID(ctx context.Context, publicID string, limit, offset int) ([]*UserSearchResult, error)
}
