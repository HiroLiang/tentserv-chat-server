package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type BindAccountInput struct {
	DeviceID string
}

type BindAccountOutput struct {
	DeviceID  string
	AccountID int64
}

type BindAccountUseCase struct {
	deviceRepo device.Repository
}

func NewBindAccountUseCase(deviceRepo device.Repository) *BindAccountUseCase {
	return &BindAccountUseCase{deviceRepo: deviceRepo}
}

func (uc *BindAccountUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[BindAccountInput],
) (BindAccountOutput, error) {
	if input.Base.Auth == nil {
		return BindAccountOutput{}, ErrUnauthorized
	}

	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return BindAccountOutput{}, ErrInvalidID
	}

	// Verify device exists
	if _, err := uc.deviceRepo.FindByID(ctx, deviceID); err != nil {
		return BindAccountOutput{}, ErrDeviceNotFound
	}

	if err := uc.deviceRepo.BindAccount(ctx, deviceID, input.Base.Auth.AccountID); err != nil {
		return BindAccountOutput{}, ErrBindFailed
	}

	return BindAccountOutput{
		DeviceID:  deviceID.String(),
		AccountID: int64(input.Base.Auth.AccountID),
	}, nil
}
