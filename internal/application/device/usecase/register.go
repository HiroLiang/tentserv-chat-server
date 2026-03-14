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

type RegisterInput struct {
	DeviceID string
	Platform string
	Name     string
}

type RegisterOutput struct {
	DeviceID  string
	Name      string
	Platform  string
	CreatedAt string
}

type RegisterUseCase struct {
	uof        transaction.UnitOfWork
	deviceRepo device.Repository
}

func NewRegisterUseCase(uof transaction.UnitOfWork, deviceRepo device.Repository) *RegisterUseCase {
	return &RegisterUseCase{
		uof:        uof,
		deviceRepo: deviceRepo,
	}
}

func (uc *RegisterUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[RegisterInput],
) (RegisterOutput, error) {
	ctx, tx, err := uc.uof.Begin(ctx)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Create Device
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return RegisterOutput{}, ErrInvalidID
	}

	platform, err := device.ParsePlatform(input.Data.Platform)
	if err != nil {
		return RegisterOutput{}, ErrInvalidPlatform
	}

	newDevice := device.NewDevice(
		deviceID,
		platform,
		input.Data.Name,
	)

	// Idempotent: return existing if already registered
	if err := uc.deviceRepo.Create(ctx, newDevice); err != nil {
		switch {
		case errors.Is(err, device.ErrDeviceAlreadyExists):
			return RegisterOutput{}, ErrDeviceExist
		default:
			return RegisterOutput{}, ErrRegisterFailed
		}
	}

	// Return the device profile
	newDevice, err = uc.deviceRepo.FindByID(ctx, deviceID)
	if err != nil {
		return RegisterOutput{}, ErrRegisterFailed
	}

	return uc.toOutput(newDevice), tx.Commit()
}

func (uc *RegisterUseCase) toOutput(d *device.Device) RegisterOutput {
	return RegisterOutput{
		DeviceID:  d.ID.String(),
		Name:      d.Name,
		Platform:  d.Platform.String(),
		CreatedAt: timeutil.Format(d.CreatedAt, timeutil.FormatISO),
	}
}
