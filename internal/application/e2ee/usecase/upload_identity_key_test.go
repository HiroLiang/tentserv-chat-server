package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadIdentityRepoStub struct {
	upsert  func(ctx context.Context, key *useridentitykey.UserIdentityKey) error
	lastKey *useridentitykey.UserIdentityKey
}

func (s *uploadIdentityRepoStub) FindByUserAndDevice(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
	return nil, useridentitykey.ErrNotFound
}

func (s *uploadIdentityRepoStub) FindByUser(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	return nil, nil
}

func (s *uploadIdentityRepoStub) Upsert(ctx context.Context, key *useridentitykey.UserIdentityKey) error {
	s.lastKey = key
	if s.upsert != nil {
		return s.upsert(ctx, key)
	}
	return nil
}

type uploadIdentityKeyVerifierStub struct {
	fingerprint string
	seenKey     []byte
}

func (s *uploadIdentityKeyVerifierStub) VerifySignedPreKey([]byte, []byte, []byte) bool {
	return false
}

func (s *uploadIdentityKeyVerifierStub) FingerprintKey(publicKey []byte) string {
	s.seenKey = append([]byte{}, publicKey...)
	return s.fingerprint
}

func TestUploadIdentityKeyUseCase_SuccessMapsDeviceAndFingerprint(t *testing.T) {
	repo := &uploadIdentityRepoStub{}
	publicKey := repeatedUploadIdentityByte(1, 32)
	signPublicKey := repeatedUploadIdentityByte(2, 32)
	sum := sha256.Sum256(publicKey)
	expectedFingerprint := hex.EncodeToString(sum[:])
	verifier := &uploadIdentityKeyVerifierStub{fingerprint: expectedFingerprint}
	uc := NewUploadIdentityKeyUseCase(repo, verifier)

	out, err := uc.Execute(context.Background(), uploadIdentityInput(
		"550e8400-e29b-41d4-a716-446655440000",
		base64.StdEncoding.EncodeToString(publicKey),
		base64.StdEncoding.EncodeToString(signPublicKey),
	))

	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, repo.lastKey)
	deviceID, parseErr := shared.ParseDeviceID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, parseErr)
	assert.Equal(t, expectedFingerprint, out.Fingerprint)
	assert.Equal(t, expectedFingerprint, string(repo.lastKey.Fingerprint))
	assert.Equal(t, shared.UserID(42), repo.lastKey.UserID)
	assert.Equal(t, deviceID, repo.lastKey.DeviceID)
	assert.Equal(t, publicKey, verifier.seenKey)
	assert.Equal(t, publicKey, repo.lastKey.PublicKey[:])
	assert.Equal(t, signPublicKey, repo.lastKey.SignPublicKey[:])
}

func TestUploadIdentityKeyUseCase_InvalidPublicKeyBase64ReturnsInvalidSignature(t *testing.T) {
	uc := NewUploadIdentityKeyUseCase(&uploadIdentityRepoStub{}, &uploadIdentityKeyVerifierStub{})

	out, err := uc.Execute(context.Background(), uploadIdentityInput(
		"550e8400-e29b-41d4-a716-446655440000",
		"not-base64",
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(2, 32)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestUploadIdentityKeyUseCase_InvalidSignKeyLengthReturnsInvalidSignature(t *testing.T) {
	uc := NewUploadIdentityKeyUseCase(&uploadIdentityRepoStub{}, &uploadIdentityKeyVerifierStub{})

	out, err := uc.Execute(context.Background(), uploadIdentityInput(
		"550e8400-e29b-41d4-a716-446655440000",
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(1, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(2, 31)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestUploadIdentityKeyUseCase_InvalidDeviceIDReturnsError(t *testing.T) {
	uc := NewUploadIdentityKeyUseCase(&uploadIdentityRepoStub{}, &uploadIdentityKeyVerifierStub{})

	out, err := uc.Execute(context.Background(), uploadIdentityInput(
		"not-a-device-id",
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(1, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(2, 32)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestUploadIdentityKeyUseCase_PropagatesUpsertError(t *testing.T) {
	repoErr := errors.New("upsert failed")
	uc := NewUploadIdentityKeyUseCase(&uploadIdentityRepoStub{
		upsert: func(context.Context, *useridentitykey.UserIdentityKey) error {
			return repoErr
		},
	}, &uploadIdentityKeyVerifierStub{fingerprint: "fp"})

	out, err := uc.Execute(context.Background(), uploadIdentityInput(
		"550e8400-e29b-41d4-a716-446655440000",
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(1, 32)),
		base64.StdEncoding.EncodeToString(repeatedUploadIdentityByte(2, 32)),
	))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func uploadIdentityInput(deviceID, publicKey, signPublicKey string) appShared.UseCaseInput[UploadIdentityKeyInput] {
	return appShared.UseCaseInput[UploadIdentityKeyInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: 42},
		},
		Data: UploadIdentityKeyInput{
			DeviceID:      deviceID,
			PublicKey:     publicKey,
			SignPublicKey: signPublicKey,
		},
	}
}

func repeatedUploadIdentityByte(value byte, length int) []byte {
	return bytes.Repeat([]byte{value}, length)
}
