package usecase

import (
	"context"
	"encoding/base64"
	"fmt"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appCrypto "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/crypto"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
)

type UploadIdentityKeyInput struct {
	DeviceID  string
	PublicKey string // base64-encoded 32-byte Curve25519 key
}

type UploadIdentityKeyOutput struct {
	Fingerprint string
}

type UploadIdentityKeyUseCase struct {
	identityKeyRepo useridentitykey.Repository
	keyVerifier     appCrypto.KeyVerifier
}

func NewUploadIdentityKeyUseCase(
	identityKeyRepo useridentitykey.Repository,
	keyVerifier appCrypto.KeyVerifier,
) *UploadIdentityKeyUseCase {
	return &UploadIdentityKeyUseCase{
		identityKeyRepo: identityKeyRepo,
		keyVerifier:     keyVerifier,
	}
}

func (u *UploadIdentityKeyUseCase) Execute(
	ctx context.Context,
	input appShared.UseCaseInput[UploadIdentityKeyInput],
) (*UploadIdentityKeyOutput, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(input.Data.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: decode public key", ErrInvalidSignature)
	}
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("%w: public key must be 32 bytes", ErrInvalidSignature)
	}

	deviceID, err := shared.ParseDeviceID(input.Data.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device id: %w", err)
	}

	fingerprint := u.keyVerifier.FingerprintKey(keyBytes)

	var pub useridentitykey.PublicKey
	copy(pub[:], keyBytes)

	key := &useridentitykey.UserIdentityKey{
		UserID:      input.Base.Auth.UserID,
		DeviceID:    deviceID,
		PublicKey:   pub,
		Fingerprint: useridentitykey.Fingerprint(fingerprint),
	}

	if err := u.identityKeyRepo.Upsert(ctx, key); err != nil {
		return nil, fmt.Errorf("upsert identity key: %w", err)
	}

	return &UploadIdentityKeyOutput{Fingerprint: fingerprint}, nil
}
