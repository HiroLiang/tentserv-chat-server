package usecase

import (
	"context"
	"errors"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type CheckKeyStatusInput struct {
	TargetUserID string
	DeviceID     string
}

type CheckKeyStatusOutput struct {
	IdentityKeyExists  bool
	SignedPreKeyExists bool
}

type CheckKeyStatusUseCase struct {
	identityKeyRepo  useridentitykey.Repository
	signedPreKeyRepo usersignedprekey.Repository
}

func NewCheckKeyStatusUseCase(
	identityKeyRepo useridentitykey.Repository,
	signedPreKeyRepo usersignedprekey.Repository,
) *CheckKeyStatusUseCase {
	return &CheckKeyStatusUseCase{
		identityKeyRepo:  identityKeyRepo,
		signedPreKeyRepo: signedPreKeyRepo,
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
			return &CheckKeyStatusOutput{IdentityKeyExists: false, SignedPreKeyExists: false}, nil
		}
		deviceID = keys[0].DeviceID
	} else {
		deviceID, err = shared.ParseDeviceID(input.Data.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("invalid device id: %w", err)
		}
	}

	out := &CheckKeyStatusOutput{}

	_, err = u.identityKeyRepo.FindByUserAndDevice(ctx, targetUserID, deviceID)
	switch {
	case err == nil:
		out.IdentityKeyExists = true
	case errors.Is(err, useridentitykey.ErrNotFound):
		out.IdentityKeyExists = false
	default:
		return nil, fmt.Errorf("find identity key: %w", err)
	}

	_, err = u.signedPreKeyRepo.FindActive(ctx, targetUserID, deviceID)
	switch {
	case err == nil:
		out.SignedPreKeyExists = true
	case errors.Is(err, usersignedprekey.ErrNotFound):
		out.SignedPreKeyExists = false
	default:
		return nil, fmt.Errorf("find active signed prekey: %w", err)
	}

	return out, nil
}
