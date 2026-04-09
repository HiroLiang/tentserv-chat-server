package usecase

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadSignedIdentityRepoStub struct {
	findByUserAndDevice func(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error)
}

func (s *uploadSignedIdentityRepoStub) FindByUserAndDevice(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
	if s.findByUserAndDevice != nil {
		return s.findByUserAndDevice(ctx, userID, deviceID)
	}
	return nil, useridentitykey.ErrNotFound
}

func (s *uploadSignedIdentityRepoStub) FindByUser(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	return nil, nil
}

func (s *uploadSignedIdentityRepoStub) Upsert(context.Context, *useridentitykey.UserIdentityKey) error {
	return nil
}

type uploadSignedPreKeyRepoStub struct {
	deactivateAll func(ctx context.Context, userID user.ID, deviceID device.ID) error
	add           func(ctx context.Context, key *usersignedprekey.UserSignedPreKey) error
	deactivated   struct {
		userID   user.ID
		deviceID device.ID
		calls    int
	}
	addedKey *usersignedprekey.UserSignedPreKey
}

func (s *uploadSignedPreKeyRepoStub) FindActive(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
	return nil, usersignedprekey.ErrNotFound
}

func (s *uploadSignedPreKeyRepoStub) FindByKeyID(context.Context, user.ID, device.ID, usersignedprekey.KeyID) (*usersignedprekey.UserSignedPreKey, error) {
	return nil, usersignedprekey.ErrNotFound
}

func (s *uploadSignedPreKeyRepoStub) Add(ctx context.Context, key *usersignedprekey.UserSignedPreKey) error {
	s.addedKey = key
	if s.add != nil {
		return s.add(ctx, key)
	}
	return nil
}

func (s *uploadSignedPreKeyRepoStub) DeactivateAll(ctx context.Context, userID user.ID, deviceID device.ID) error {
	s.deactivated.userID = userID
	s.deactivated.deviceID = deviceID
	s.deactivated.calls++
	if s.deactivateAll != nil {
		return s.deactivateAll(ctx, userID, deviceID)
	}
	return nil
}

type uploadSignedVerifierStub struct {
	verifyResult    bool
	identityKeySeen []byte
	publicKeySeen   []byte
	signatureSeen   []byte
}

func (s *uploadSignedVerifierStub) VerifySignedPreKey(identityKey, publicKey, signature []byte) bool {
	s.identityKeySeen = append([]byte{}, identityKey...)
	s.publicKeySeen = append([]byte{}, publicKey...)
	s.signatureSeen = append([]byte{}, signature...)
	return s.verifyResult
}

func (s *uploadSignedVerifierStub) FingerprintKey([]byte) string {
	return ""
}

func TestUploadSignedPreKeyUseCase_SuccessUsesDeviceScopedIdentityAndStoresKey(t *testing.T) {
	logger.InitTestEnv()
	deviceIDText := "550e8400-e29b-41d4-a716-446655440000"
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	require.NoError(t, err)

	var signPublicKey useridentitykey.SignPublicKey
	copy(signPublicKey[:], repeatedUploadSignedByte(1, 32))
	publicKey := repeatedUploadSignedByte(2, 32)
	signature := repeatedUploadSignedByte(3, 64)
	verifier := &uploadSignedVerifierStub{verifyResult: true}
	signedRepo := &uploadSignedPreKeyRepoStub{}
	uc := NewUploadSignedPreKeyUseCase(
		&uploadSignedIdentityRepoStub{
			findByUserAndDevice: func(_ context.Context, gotUserID user.ID, gotDeviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
				require.Equal(t, user.ID(42), gotUserID)
				require.Equal(t, deviceID, gotDeviceID)
				return &useridentitykey.UserIdentityKey{
					UserID:        42,
					DeviceID:      deviceID,
					SignPublicKey: signPublicKey,
				}, nil
			},
		},
		signedRepo,
		verifier,
	)

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		deviceIDText,
		7,
		base64.StdEncoding.EncodeToString(publicKey),
		base64.StdEncoding.EncodeToString(signature),
	))

	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, signedRepo.addedKey)
	assert.Equal(t, 1, signedRepo.deactivated.calls)
	assert.Equal(t, user.ID(42), signedRepo.deactivated.userID)
	assert.Equal(t, deviceID, signedRepo.deactivated.deviceID)
	assert.Equal(t, repeatedUploadSignedByte(1, 32), verifier.identityKeySeen)
	assert.Equal(t, publicKey, verifier.publicKeySeen)
	assert.Equal(t, signature, verifier.signatureSeen)
	assert.Equal(t, user.ID(42), signedRepo.addedKey.UserID)
	assert.Equal(t, deviceID, signedRepo.addedKey.DeviceID)
	assert.Equal(t, usersignedprekey.KeyID(7), signedRepo.addedKey.KeyID)
	assert.Equal(t, publicKey, signedRepo.addedKey.PublicKey[:])
	assert.Equal(t, signature, signedRepo.addedKey.Signature[:])
	assert.True(t, signedRepo.addedKey.IsActive)
}

func TestUploadSignedPreKeyUseCase_InvalidDeviceIDReturnsError(t *testing.T) {
	logger.InitTestEnv()
	uc := NewUploadSignedPreKeyUseCase(&uploadSignedIdentityRepoStub{}, &uploadSignedPreKeyRepoStub{}, &uploadSignedVerifierStub{})

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"not-a-device-id",
		1,
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(2, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestUploadSignedPreKeyUseCase_InvalidPublicKeyReturnsInvalidSignature(t *testing.T) {
	logger.InitTestEnv()
	uc := NewUploadSignedPreKeyUseCase(&uploadSignedIdentityRepoStub{}, &uploadSignedPreKeyRepoStub{}, &uploadSignedVerifierStub{})

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"550e8400-e29b-41d4-a716-446655440000",
		1,
		"not-base64",
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestUploadSignedPreKeyUseCase_IdentityLookupFailureReturnsIdentityNotFound(t *testing.T) {
	logger.InitTestEnv()
	repoErr := errors.New("identity repo unavailable")
	uc := NewUploadSignedPreKeyUseCase(
		&uploadSignedIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return nil, repoErr
			},
		},
		&uploadSignedPreKeyRepoStub{},
		&uploadSignedVerifierStub{},
	)

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"550e8400-e29b-41d4-a716-446655440000",
		1,
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(2, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrIdentityNotFound)
}

func TestUploadSignedPreKeyUseCase_InvalidSignatureReturnsInvalidSignature(t *testing.T) {
	logger.InitTestEnv()
	var signPublicKey useridentitykey.SignPublicKey
	copy(signPublicKey[:], repeatedUploadSignedByte(1, 32))
	signedRepo := &uploadSignedPreKeyRepoStub{}
	uc := NewUploadSignedPreKeyUseCase(
		&uploadSignedIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{SignPublicKey: signPublicKey}, nil
			},
		},
		signedRepo,
		&uploadSignedVerifierStub{verifyResult: false},
	)

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"550e8400-e29b-41d4-a716-446655440000",
		1,
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(2, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
	assert.Zero(t, signedRepo.deactivated.calls)
	assert.Nil(t, signedRepo.addedKey)
}

func TestUploadSignedPreKeyUseCase_PropagatesDeactivateError(t *testing.T) {
	logger.InitTestEnv()
	var signPublicKey useridentitykey.SignPublicKey
	copy(signPublicKey[:], repeatedUploadSignedByte(1, 32))
	repoErr := errors.New("deactivate failed")
	uc := NewUploadSignedPreKeyUseCase(
		&uploadSignedIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{SignPublicKey: signPublicKey}, nil
			},
		},
		&uploadSignedPreKeyRepoStub{
			deactivateAll: func(context.Context, user.ID, device.ID) error {
				return repoErr
			},
		},
		&uploadSignedVerifierStub{verifyResult: true},
	)

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"550e8400-e29b-41d4-a716-446655440000",
		1,
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(2, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestUploadSignedPreKeyUseCase_PropagatesAddError(t *testing.T) {
	logger.InitTestEnv()
	var signPublicKey useridentitykey.SignPublicKey
	copy(signPublicKey[:], repeatedUploadSignedByte(1, 32))
	repoErr := errors.New("add failed")
	uc := NewUploadSignedPreKeyUseCase(
		&uploadSignedIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{SignPublicKey: signPublicKey}, nil
			},
		},
		&uploadSignedPreKeyRepoStub{
			add: func(context.Context, *usersignedprekey.UserSignedPreKey) error {
				return repoErr
			},
		},
		&uploadSignedVerifierStub{verifyResult: true},
	)

	out, err := uc.Execute(context.Background(), uploadSignedInput(
		"550e8400-e29b-41d4-a716-446655440000",
		1,
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(2, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadSignedByte(3, 64)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func uploadSignedInput(deviceID string, keyID uint32, publicKey, signature string) appShared.UseCaseInput[UploadSignedPreKeyInput] {
	return appShared.UseCaseInput[UploadSignedPreKeyInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: 42},
		},
		Data: UploadSignedPreKeyInput{
			DeviceID:  deviceID,
			KeyID:     keyID,
			PublicKey: publicKey,
			Signature: signature,
		},
	}
}

func repeatedUploadSignedByte(value byte, length int) []byte {
	return bytes.Repeat([]byte{value}, length)
}
