package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type otpUseCaseRepoStub struct {
	addBatch       func(ctx context.Context, keys []*userotpprekey.UserOTPPreKey) error
	countAvailable func(ctx context.Context, userID user.ID, deviceID device.ID) (int, error)
	consumeOne     func(ctx context.Context, userID user.ID, deviceID device.ID) (*userotpprekey.UserOTPPreKey, error)
	addedKeys      []*userotpprekey.UserOTPPreKey
}

func (s *otpUseCaseRepoStub) ConsumeOne(ctx context.Context, userID user.ID, deviceID device.ID) (*userotpprekey.UserOTPPreKey, error) {
	if s.consumeOne != nil {
		return s.consumeOne(ctx, userID, deviceID)
	}
	return nil, userotpprekey.ErrPoolEmpty
}

func (s *otpUseCaseRepoStub) AddBatch(ctx context.Context, keys []*userotpprekey.UserOTPPreKey) error {
	s.addedKeys = append(s.addedKeys, keys...)
	if s.addBatch != nil {
		return s.addBatch(ctx, keys)
	}
	return nil
}

func (s *otpUseCaseRepoStub) CountAvailable(ctx context.Context, userID user.ID, deviceID device.ID) (int, error) {
	if s.countAvailable != nil {
		return s.countAvailable(ctx, userID, deviceID)
	}
	return len(s.addedKeys), nil
}

func TestUploadOTPPreKeysUseCase_SuccessMapsKeysAndReturnsCount(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	repo := &otpUseCaseRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 7, nil
		},
	}
	uc := NewUploadOTPPreKeysUseCase(repo)

	out, err := uc.Execute(context.Background(), uploadOTPInput(deviceID, []OTPPreKeyItem{{
		KeyID:     3,
		PublicKey: base64.StdEncoding.EncodeToString(otpUseCaseBytesOf(9, 32)),
	}}))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 7, out.Count)
	require.Len(t, repo.addedKeys, 1)
	assert.Equal(t, shared.UserID(42), repo.addedKeys[0].UserID)
	assert.Equal(t, userotpprekey.KeyID(3), repo.addedKeys[0].KeyID)
}

func TestUploadOTPPreKeysUseCase_InvalidBase64ReturnsInvalidSignature(t *testing.T) {
	uc := NewUploadOTPPreKeysUseCase(&otpUseCaseRepoStub{})

	out, err := uc.Execute(context.Background(), uploadOTPInput("550e8400-e29b-41d4-a716-446655440000", []OTPPreKeyItem{{
		KeyID:     1,
		PublicKey: "not-base64",
	}}))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestUploadOTPPreKeysUseCase_InvalidKeyLengthReturnsInvalidSignature(t *testing.T) {
	uc := NewUploadOTPPreKeysUseCase(&otpUseCaseRepoStub{})

	out, err := uc.Execute(context.Background(), uploadOTPInput("550e8400-e29b-41d4-a716-446655440000", []OTPPreKeyItem{{
		KeyID:     1,
		PublicKey: base64.StdEncoding.EncodeToString(otpUseCaseBytesOf(1, 31)),
	}}))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestUploadOTPPreKeysUseCase_InvalidDeviceIDReturnsError(t *testing.T) {
	uc := NewUploadOTPPreKeysUseCase(&otpUseCaseRepoStub{})

	out, err := uc.Execute(context.Background(), uploadOTPInput("not-a-device-id", []OTPPreKeyItem{{
		KeyID:     1,
		PublicKey: base64.StdEncoding.EncodeToString(otpUseCaseBytesOf(1, 32)),
	}}))

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestUploadOTPPreKeysUseCase_PropagatesAddBatchError(t *testing.T) {
	repoErr := errors.New("add failed")
	uc := NewUploadOTPPreKeysUseCase(&otpUseCaseRepoStub{
		addBatch: func(context.Context, []*userotpprekey.UserOTPPreKey) error {
			return repoErr
		},
	})

	out, err := uc.Execute(context.Background(), uploadOTPInput("550e8400-e29b-41d4-a716-446655440000", []OTPPreKeyItem{{
		KeyID:     1,
		PublicKey: base64.StdEncoding.EncodeToString(otpUseCaseBytesOf(1, 32)),
	}}))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestUploadOTPPreKeysUseCase_PropagatesCountError(t *testing.T) {
	repoErr := errors.New("count failed")
	uc := NewUploadOTPPreKeysUseCase(&otpUseCaseRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 0, repoErr
		},
	})

	out, err := uc.Execute(context.Background(), uploadOTPInput("550e8400-e29b-41d4-a716-446655440000", []OTPPreKeyItem{{
		KeyID:     1,
		PublicKey: base64.StdEncoding.EncodeToString(otpUseCaseBytesOf(1, 32)),
	}}))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCountOTPPreKeysUseCase_Success(t *testing.T) {
	uc := NewCountOTPPreKeysUseCase(&otpUseCaseRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 4, nil
		},
	})

	out, err := uc.Execute(context.Background(), countOTPInput("550e8400-e29b-41d4-a716-446655440000"))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 4, out.Count)
}

func TestCountOTPPreKeysUseCase_InvalidDeviceIDReturnsError(t *testing.T) {
	uc := NewCountOTPPreKeysUseCase(&otpUseCaseRepoStub{})

	out, err := uc.Execute(context.Background(), countOTPInput("not-a-device-id"))

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestCountOTPPreKeysUseCase_PropagatesRepoError(t *testing.T) {
	repoErr := errors.New("count failed")
	uc := NewCountOTPPreKeysUseCase(&otpUseCaseRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 0, repoErr
		},
	})

	out, err := uc.Execute(context.Background(), countOTPInput("550e8400-e29b-41d4-a716-446655440000"))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func uploadOTPInput(deviceID string, keys []OTPPreKeyItem) appShared.UseCaseInput[UploadOTPPreKeysInput] {
	return appShared.UseCaseInput[UploadOTPPreKeysInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: 42},
		},
		Data: UploadOTPPreKeysInput{
			DeviceID: deviceID,
			Keys:     keys,
		},
	}
}

func countOTPInput(deviceID string) appShared.UseCaseInput[CountOTPPreKeysInput] {
	return appShared.UseCaseInput[CountOTPPreKeysInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{UserID: 42},
		},
		Data: CountOTPPreKeysInput{DeviceID: deviceID},
	}
}

func otpUseCaseBytesOf(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}
