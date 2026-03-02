package device

import (
	"context"
	"errors"

	"github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/shared/timeutil"
)

type UseCase struct {
	deviceRepo device.Repository
}

func NewUseCase(repo device.Repository) *UseCase {
	return &UseCase{deviceRepo: repo}
}

func (u *UseCase) RegisterDevice(ctx context.Context, input shared.UseCaseInput[RegisterDeviceInput]) (DeviceOutput, error) {
	platform, err := device.ParsePlatform(input.Data.Platform)
	if err != nil {
		return DeviceOutput{}, err
	}

	deviceID := device.ID(input.Data.DeviceID)

	// Idempotent: return existing if already registered
	existing, err := u.deviceRepo.FindByID(ctx, deviceID)
	if err == nil {
		return toOutput(existing), nil
	}
	if !errors.Is(err, device.ErrDeviceNotFound) {
		return DeviceOutput{}, err
	}

	d := device.NewDevice(deviceID, platform, input.Data.Name)
	if err := u.deviceRepo.Create(ctx, d); err != nil {
		return DeviceOutput{}, err
	}

	created, err := u.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return DeviceOutput{}, err
	}

	return toOutput(created), nil
}

func (u *UseCase) GetDevice(ctx context.Context, input shared.UseCaseInput[GetDeviceInput]) (DeviceOutput, error) {
	d, err := u.deviceRepo.FindByID(ctx, device.ID(input.Data.DeviceID))
	if err != nil {
		return DeviceOutput{}, device.ErrDeviceNotFound
	}

	return toOutput(d), nil
}

func (u *UseCase) UpdateDevice(ctx context.Context, input shared.UseCaseInput[UpdateDeviceInput]) (DeviceOutput, error) {
	platform, err := device.ParsePlatform(input.Data.Platform)
	if err != nil {
		return DeviceOutput{}, err
	}

	deviceID := device.ID(input.Data.DeviceID)

	d, err := u.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return DeviceOutput{}, err
	}

	d.Name = input.Data.Name
	d.Platform = platform

	if err := u.deviceRepo.Update(ctx, d); err != nil {
		return DeviceOutput{}, err
	}

	updated, err := u.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return DeviceOutput{}, err
	}

	return toOutput(updated), nil
}

func toOutput(d *device.Device) DeviceOutput {
	return DeviceOutput{
		DeviceID:  string(d.ID),
		Name:      d.Name,
		Platform:  string(d.Platform),
		CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
	}
}
