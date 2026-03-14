package usecase

import (
	"context"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/shared/timeutil"
)

type UpdateDeviceInput struct {
	Name     string
	Platform string
}

type UpdateDeviceOutput struct {
	DeviceID  string
	Name      string
	Platform  string
	CreatedAt string
}

type UpdateDeviceUseCase struct {
	deviceRepo device.Repository
}

func NewUpdateDeviceUseCase(deviceRepo device.Repository) *UpdateDeviceUseCase {
	return &UpdateDeviceUseCase{
		deviceRepo: deviceRepo,
	}
}

func (uc *UpdateDeviceUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UpdateDeviceInput],
) (*UpdateDeviceOutput, error) {

	// Find Device
	deviceData, err := uc.deviceRepo.FindByID(ctx, input.Base.Request.DeviceID)
	if err != nil {
		return nil, ErrDeviceNotFound
	}

	// Parse Platform
	platform, err := device.ParsePlatform(input.Data.Platform)
	if err != nil {
		return nil, ErrInvalidPlatform
	}

	deviceData.Name = input.Data.Name
	deviceData.Platform = platform

	if err := uc.deviceRepo.Update(ctx, deviceData); err != nil {
		return nil, ErrUpdateFailed
	}

	return toUpdateDeviceOutput(deviceData), nil
}

func toUpdateDeviceOutput(d *device.Device) *UpdateDeviceOutput {
	return &UpdateDeviceOutput{
		DeviceID:  d.ID.String(),
		Name:      d.Name,
		Platform:  d.Platform.String(),
		CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
	}
}
