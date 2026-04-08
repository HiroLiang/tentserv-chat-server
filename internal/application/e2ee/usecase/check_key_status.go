package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type CheckKeyStatusInput struct {
	TargetUserID string
	DeviceID     string
}

type CheckKeyStatusOutput struct {
	IdentityKeyExists  bool
	SignedPreKeyExists bool
	DeviceID           string
	IdentityKey        string
	IdentityKeySign    string
	SignedPreKey       string
	SPKSignature       string
	SPKKeyID           uint32
	OTPPreKeyCount     int
}

type CheckKeyStatusUseCase struct {
	identityKeyRepo  useridentitykey.Repository
	signedPreKeyRepo usersignedprekey.Repository
	otpPreKeyRepo    userotpprekey.Repository
}

func NewCheckKeyStatusUseCase(
	identityKeyRepo useridentitykey.Repository,
	signedPreKeyRepo usersignedprekey.Repository,
	otpPreKeyRepo userotpprekey.Repository,
) *CheckKeyStatusUseCase {
	return &CheckKeyStatusUseCase{
		identityKeyRepo:  identityKeyRepo,
		signedPreKeyRepo: signedPreKeyRepo,
		otpPreKeyRepo:    otpPreKeyRepo,
	}
}

func (u *CheckKeyStatusUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[CheckKeyStatusInput],
) (*CheckKeyStatusOutput, error) {
	targetUserID, err := shared.ParseUserID(input.Data.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid target user id: %w", err)
	}

	var deviceID shared.DeviceID
	if input.Data.DeviceID == "" {
		keys, err := u.identityKeyRepo.FindByUser(ctx, targetUserID)
		if err != nil {
			return nil, fmt.Errorf("find identity keys by user: %w", err)
		}
		if len(keys) == 0 {
			return &CheckKeyStatusOutput{IdentityKeyExists: false, SignedPreKeyExists: false, OTPPreKeyCount: 0}, nil
		}
		deviceID = keys[0].DeviceID
	} else {
		deviceID, err = shared.ParseDeviceID(input.Data.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("invalid device id: %w", err)
		}
	}

	out := &CheckKeyStatusOutput{}

	identityKey, err := u.identityKeyRepo.FindByUserAndDevice(ctx, targetUserID, deviceID)
	switch {
	case err == nil:
		out.IdentityKeyExists = true
		out.DeviceID = deviceID.String()
		out.IdentityKey = base64.StdEncoding.EncodeToString(identityKey.PublicKey[:])
		out.IdentityKeySign = base64.StdEncoding.EncodeToString(identityKey.SignPublicKey[:])
	case errors.Is(err, useridentitykey.ErrNotFound):
		out.IdentityKeyExists = false
	default:
		return nil, fmt.Errorf("find identity key: %w", err)
	}

	signedPreKey, err := u.signedPreKeyRepo.FindActive(ctx, targetUserID, deviceID)
	switch {
	case err == nil:
		out.SignedPreKeyExists = true
		out.SignedPreKey = base64.StdEncoding.EncodeToString(signedPreKey.PublicKey[:])
		out.SPKSignature = base64.StdEncoding.EncodeToString(signedPreKey.Signature[:])
		out.SPKKeyID = uint32(signedPreKey.KeyID)
	case errors.Is(err, usersignedprekey.ErrNotFound):
		out.SignedPreKeyExists = false
	default:
		return nil, fmt.Errorf("find active signed prekey: %w", err)
	}

	count, err := u.otpPreKeyRepo.CountAvailable(ctx, targetUserID, deviceID)
	if err != nil {
		return nil, fmt.Errorf("count otp prekeys: %w", err)
	}
	out.OTPPreKeyCount = count

	return out, nil
}
