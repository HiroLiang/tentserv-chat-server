package device

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type Repository interface {
	FindByID(ctx context.Context, deviceID ID) (*Device, error)
	FindAllByUserID(ctx context.Context, userID user.ID) ([]*Device, error)
	Create(ctx context.Context, d *Device) error
	Update(ctx context.Context, d *Device) error
	BindUser(ctx context.Context, deviceID ID, userID user.ID) error
}
