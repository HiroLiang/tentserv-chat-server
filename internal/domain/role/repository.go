package role

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, id shared.RoleID) (*Role, error)
	FindByCode(ctx context.Context, code Code) (*Role, error)
	FindAll(ctx context.Context) ([]*Role, error)
	Create(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
}
