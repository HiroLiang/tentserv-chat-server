package device

import (
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

func toDomain(r *DeviceRecord) (*device.Device, error) {
	platform, err := device.ParsePlatform(r.Platform)
	if err != nil {
		return nil, err
	}
	deviceID, err := shared.ParseDeviceID(r.ID)
	if err != nil {
		return nil, err
	}
	return &device.Device{
		ID:        deviceID,
		Platform:  platform,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}, nil
}

func toRecord(d *device.Device) *DeviceRecord {
	return &DeviceRecord{
		ID:       d.ID.String(),
		Platform: string(d.Platform),
		Name:     d.Name,
	}
}
