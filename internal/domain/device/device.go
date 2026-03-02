package device

import "time"

type Device struct {
	ID        ID
	Platform  Platform
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewDevice(id ID, platform Platform, name string) *Device {
	return &Device{
		ID:       id,
		Platform: platform,
		Name:     name,
	}
}
