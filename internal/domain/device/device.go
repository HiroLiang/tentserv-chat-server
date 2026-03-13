package device

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Device struct {
	ID        shared.DeviceID
	Platform  Platform
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewDevice(
	id shared.DeviceID,
	platform Platform,
	name string,
) *Device {
	return &Device{
		ID:       id,
		Platform: platform,
		Name:     name,
	}
}
