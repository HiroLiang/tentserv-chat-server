package usecase

import (
	"context"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type DeleteDeviceInput struct {
	DeviceID string
}

type DeleteDeviceUseCase struct {
	deviceRepo device.Repository
}

func NewDeleteDeviceUseCase(deviceRepo device.Repository) *DeleteDeviceUseCase {
	return &DeleteDeviceUseCase{deviceRepo: deviceRepo}
}

func (uc *DeleteDeviceUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[DeleteDeviceInput],
) error {
	if input.Base.Auth == nil {
		return ErrUnauthorized
	}

	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return ErrInvalidID
	}

	if err := uc.deviceRepo.DeleteByAccount(ctx, deviceID, input.Base.Auth.AccountID); err != nil {
		return ErrDeviceNotFound
	}

	return nil
}
