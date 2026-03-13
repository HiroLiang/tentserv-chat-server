package usecase

import (
	"context"
	"errors"

	appShared "github.com/HiroLiang/goat-server/internal/application/shared"
	"github.com/HiroLiang/goat-server/internal/domain/device"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/domain/transaction"
	"github.com/HiroLiang/goat-server/internal/shared/timeutil"
)

type GetProfileInput struct {
	DeviceID string
}

type GetProfileOutput struct {
	DeviceID  string
	Name      string
	Platform  string
	CreatedAt string
}

type GetProfileUseCase struct {
	uof        transaction.UnitOfWork
	deviceRepo device.Repository
}

func (uc *GetProfileUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetProfileInput],
) (GetProfileOutput, error) {
	deviceId, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return GetProfileOutput{}, ErrInvalidID
	}

	deviceData, err := uc.deviceRepo.FindByID(ctx, deviceId)
	if err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceNotFound):
			return GetProfileOutput{}, ErrDeviceNotFound
		default:
			return GetProfileOutput{}, err
		}
	}

	return uc.toOutput(deviceData), nil
}

func (uc *GetProfileUseCase) toOutput(d *device.Device) GetProfileOutput {
	return GetProfileOutput{
		DeviceID:  d.ID.String(),
		Name:      d.Name,
		Platform:  string(d.Platform),
		CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
	}
}
