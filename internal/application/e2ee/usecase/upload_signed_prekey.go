package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appCrypto "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/crypto"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type UploadSignedPreKeyInput struct {
	DeviceID  string
	KeyID     uint32
	PublicKey string // base64
	Signature string // base64
}

type UploadSignedPreKeyOutput struct{}

type UploadSignedPreKeyUseCase struct {
	identityKeyRepo  useridentitykey.Repository
	signedPreKeyRepo usersignedprekey.Repository
	keyVerifier      appCrypto.KeyVerifier
}

func NewUploadSignedPreKeyUseCase(
	identityKeyRepo useridentitykey.Repository,
	signedPreKeyRepo usersignedprekey.Repository,
	keyVerifier appCrypto.KeyVerifier,
) *UploadSignedPreKeyUseCase {
	return &UploadSignedPreKeyUseCase{
		identityKeyRepo:  identityKeyRepo,
		signedPreKeyRepo: signedPreKeyRepo,
		keyVerifier:      keyVerifier,
	}
}

func (u *UploadSignedPreKeyUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UploadSignedPreKeyInput],
) (*UploadSignedPreKeyOutput, error) {
	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device id: %w", err)
	}

	pubBytes, err := base64.StdEncoding.DecodeString(input.Data.PublicKey)
	if err != nil || len(pubBytes) != 32 {
		return nil, fmt.Errorf("%w: decode public key", ErrInvalidSignature)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(input.Data.Signature)
	if err != nil || len(sigBytes) != 64 {
		return nil, fmt.Errorf("%w: decode signature", ErrInvalidSignature)
	}

	identityKey, err := u.identityKeyRepo.FindByUserAndDevice(ctx, input.Base.Auth.UserID, deviceID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	if !u.keyVerifier.VerifySignedPreKey(identityKey.PublicKey[:], pubBytes, sigBytes) {
		return nil, ErrInvalidSignature
	}

	if err := u.signedPreKeyRepo.DeactivateAll(ctx, input.Base.Auth.UserID, deviceID); err != nil {
		return nil, fmt.Errorf("deactivate signed prekeys: %w", err)
	}

	var pub usersignedprekey.PublicKey
	copy(pub[:], pubBytes)
	var sig usersignedprekey.Signature
	copy(sig[:], sigBytes)

	key := &usersignedprekey.UserSignedPreKey{
		UserID:    input.Base.Auth.UserID,
		DeviceID:  deviceID,
		KeyID:     usersignedprekey.KeyID(input.Data.KeyID),
		PublicKey: pub,
		Signature: sig,
		IsActive:  true,
	}

	if err := u.signedPreKeyRepo.Add(ctx, key); err != nil {
		return nil, fmt.Errorf("add signed prekey: %w", err)
	}

	return &UploadSignedPreKeyOutput{}, nil
}
