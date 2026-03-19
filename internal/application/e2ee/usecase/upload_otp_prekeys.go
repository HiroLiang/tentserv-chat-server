package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
)

type OTPPreKeyItem struct {
	KeyID     uint32
	PublicKey string // base64
}

type UploadOTPPreKeysInput struct {
	DeviceID string
	Keys     []OTPPreKeyItem
}

type UploadOTPPreKeysOutput struct {
	Count int
}

type UploadOTPPreKeysUseCase struct {
	otpPreKeyRepo userotpprekey.Repository
}

func NewUploadOTPPreKeysUseCase(otpPreKeyRepo userotpprekey.Repository) *UploadOTPPreKeysUseCase {
	return &UploadOTPPreKeysUseCase{otpPreKeyRepo: otpPreKeyRepo}
}

func (u *UploadOTPPreKeysUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UploadOTPPreKeysInput],
) (*UploadOTPPreKeysOutput, error) {
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device id: %w", err)
	}

	keys := make([]*userotpprekey.UserOTPPreKey, 0, len(input.Data.Keys))
	for _, item := range input.Data.Keys {
		pubBytes, err := base64.StdEncoding.DecodeString(item.PublicKey)
		if err != nil || len(pubBytes) != 32 {
			return nil, fmt.Errorf("%w: decode otp prekey", ErrInvalidSignature)
		}
		var pub userotpprekey.PublicKey
		copy(pub[:], pubBytes)

		keys = append(keys, &userotpprekey.UserOTPPreKey{
			UserID:    input.Base.Auth.UserID,
			DeviceID:  deviceID,
			KeyID:     userotpprekey.KeyID(item.KeyID),
			PublicKey: pub,
		})
	}

	if err := u.otpPreKeyRepo.AddBatch(ctx, keys); err != nil {
		return nil, fmt.Errorf("add otp prekeys: %w", err)
	}

	return &UploadOTPPreKeysOutput{Count: len(keys)}, nil
}
