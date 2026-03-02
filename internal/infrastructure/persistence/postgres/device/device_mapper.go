package device

import (
	"github.com/HiroLiang/goat-server/internal/domain/device"
)

func toDomain(r *DeviceRecord) (*device.Device, error) {
	platform, err := device.ParsePlatform(r.Platform)
	if err != nil {
		return nil, err
	}
	return &device.Device{
		ID:        device.ID(r.ID),
		Platform:  platform,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func toRecord(d *device.Device) *DeviceRecord {
	return &DeviceRecord{
		ID:       string(d.ID),
		Platform: string(d.Platform),
		Name:     d.Name,
	}
}
