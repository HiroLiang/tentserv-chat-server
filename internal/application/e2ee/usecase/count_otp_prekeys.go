package usecase

import (
	"context"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
)

type CountOTPPreKeysInput struct {
	DeviceID string
}

type CountOTPPreKeysOutput struct {
	Count int
}

type CountOTPPreKeysUseCase struct {
	otpPreKeyRepo userotpprekey.Repository
}

func NewCountOTPPreKeysUseCase(otpPreKeyRepo userotpprekey.Repository) *CountOTPPreKeysUseCase {
	return &CountOTPPreKeysUseCase{otpPreKeyRepo: otpPreKeyRepo}
}

func (u *CountOTPPreKeysUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[CountOTPPreKeysInput],
) (*CountOTPPreKeysOutput, error) {
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device id: %w", err)
	}

	count, err := u.otpPreKeyRepo.CountAvailable(ctx, input.Base.Auth.UserID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("count otp prekeys: %w", err)
	}

	return &CountOTPPreKeysOutput{Count: count}, nil
}
