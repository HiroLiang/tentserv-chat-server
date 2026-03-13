package device

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, deviceID shared.DeviceID) (*Device, error)
	FindAllByUserID(ctx context.Context, userID shared.UserID) ([]*Device, error)
	Create(ctx context.Context, d *Device) error
	Update(ctx context.Context, d *Device) error
}
