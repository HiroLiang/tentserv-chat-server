package device

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindByID(ctx context.Context, deviceID shared.DeviceID) (*Device, error)
	FindAllByAccountID(ctx context.Context, accountID shared.AccountID) ([]*Device, error)
	Create(ctx context.Context, d *Device) error
	Update(ctx context.Context, d *Device) error
	BindAccount(ctx context.Context, deviceID shared.DeviceID, accountID shared.AccountID) error
	DeleteByAccount(ctx context.Context, deviceID shared.DeviceID, accountID shared.AccountID) error
}
