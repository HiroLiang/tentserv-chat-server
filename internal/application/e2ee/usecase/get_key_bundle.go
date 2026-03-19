package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type GetKeyBundleInput struct {
	TargetUserID string
	DeviceID     string
}

type KeyBundleOutput struct {
	IdentityKey  string // base64
	SignedPreKey string // base64
	SPKSignature string // base64
	SPKKeyID     uint32
	OTPPreKey    *string // base64, optional
	OTPPreKeyID  *uint32 // optional
}

type GetKeyBundleUseCase struct {
	identityKeyRepo  useridentitykey.Repository
	signedPreKeyRepo usersignedprekey.Repository
	otpPreKeyRepo    userotpprekey.Repository
	dispatcher       appPush.Dispatcher
}

func NewGetKeyBundleUseCase(
	identityKeyRepo useridentitykey.Repository,
	signedPreKeyRepo usersignedprekey.Repository,
	otpPreKeyRepo userotpprekey.Repository,
	dispatcher appPush.Dispatcher,
) *GetKeyBundleUseCase {
	return &GetKeyBundleUseCase{
		identityKeyRepo:  identityKeyRepo,
		signedPreKeyRepo: signedPreKeyRepo,
		otpPreKeyRepo:    otpPreKeyRepo,
		dispatcher:       dispatcher,
	}
}

func (u *GetKeyBundleUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[GetKeyBundleInput],
) (*KeyBundleOutput, error) {
	targetUserID, err := shared.ParseUserID(input.Data.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid target user id: %w", err)
	}

	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device id: %w", err)
	}

	identityKey, err := u.identityKeyRepo.FindByUserAndDevice(ctx, targetUserID, deviceID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	signedPreKey, err := u.signedPreKeyRepo.FindActive(ctx, targetUserID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("find active signed prekey: %w", err)
	}

	out := &KeyBundleOutput{
		IdentityKey:  base64.StdEncoding.EncodeToString(identityKey.PublicKey[:]),
		SignedPreKey: base64.StdEncoding.EncodeToString(signedPreKey.PublicKey[:]),
		SPKSignature: base64.StdEncoding.EncodeToString(signedPreKey.Signature[:]),
		SPKKeyID:     uint32(signedPreKey.KeyID),
	}

	otpKey, err := u.otpPreKeyRepo.ConsumeOne(ctx, targetUserID, deviceID)
	if err == nil && otpKey != nil {
		encoded := base64.StdEncoding.EncodeToString(otpKey.PublicKey[:])
		keyID := uint32(otpKey.KeyID)
		out.OTPPreKey = &encoded
		out.OTPPreKeyID = &keyID
	}

	// Check remaining OTP prekeys; dispatch replenish request if below threshold
	remaining, err := u.otpPreKeyRepo.CountAvailable(ctx, targetUserID, deviceID)
	if err == nil && remaining < OTPReplenishThreshold {
		payload, _ := json.Marshal(map[string]interface{}{
			"user_id":   targetUserID,
			"device_id": deviceID.String(),
		})
		_ = u.dispatcher.Dispatch(ctx, targetUserID, string(deliveryqueue.PayloadTypeReplenishOTP), payload)
	}

	return out, nil
}
