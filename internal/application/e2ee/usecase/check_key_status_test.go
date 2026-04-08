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
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkKeyStatusIdentityRepoStub struct {
	findByUserAndDevice func(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error)
	findByUser          func(ctx context.Context, userID user.ID) ([]*useridentitykey.UserIdentityKey, error)
}

func (s *checkKeyStatusIdentityRepoStub) FindByUserAndDevice(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
	if s.findByUserAndDevice != nil {
		return s.findByUserAndDevice(ctx, userID, deviceID)
	}
	return nil, useridentitykey.ErrNotFound
}

func (s *checkKeyStatusIdentityRepoStub) FindByUser(ctx context.Context, userID user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	if s.findByUser != nil {
		return s.findByUser(ctx, userID)
	}
	return nil, nil
}

func (s *checkKeyStatusIdentityRepoStub) Upsert(context.Context, *useridentitykey.UserIdentityKey) error {
	return nil
}

type checkKeyStatusSignedPreKeyRepoStub struct {
	findActive func(ctx context.Context, userID user.ID, deviceID device.ID) (*usersignedprekey.UserSignedPreKey, error)
}

func (s *checkKeyStatusSignedPreKeyRepoStub) FindActive(ctx context.Context, userID user.ID, deviceID device.ID) (*usersignedprekey.UserSignedPreKey, error) {
	if s.findActive != nil {
		return s.findActive(ctx, userID, deviceID)
	}
	return nil, usersignedprekey.ErrNotFound
}

func (s *checkKeyStatusSignedPreKeyRepoStub) FindByKeyID(context.Context, user.ID, device.ID, usersignedprekey.KeyID) (*usersignedprekey.UserSignedPreKey, error) {
	return nil, usersignedprekey.ErrNotFound
}

func (s *checkKeyStatusSignedPreKeyRepoStub) Add(context.Context, *usersignedprekey.UserSignedPreKey) error {
	return nil
}

func (s *checkKeyStatusSignedPreKeyRepoStub) DeactivateAll(context.Context, user.ID, device.ID) error {
	return nil
}

type checkKeyStatusOTPPreKeyRepoStub struct {
	count          int
	consumeCalls   int
	countAvailable func(ctx context.Context, userID user.ID, deviceID device.ID) (int, error)
}

func (s *checkKeyStatusOTPPreKeyRepoStub) ConsumeOne(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
	s.consumeCalls++
	return nil, userotpprekey.ErrPoolEmpty
}

func (s *checkKeyStatusOTPPreKeyRepoStub) AddBatch(context.Context, []*userotpprekey.UserOTPPreKey) error {
	return nil
}

func (s *checkKeyStatusOTPPreKeyRepoStub) CountAvailable(ctx context.Context, userID user.ID, deviceID device.ID) (int, error) {
	if s.countAvailable != nil {
		return s.countAvailable(ctx, userID, deviceID)
	}
	return s.count, nil
}

func newCheckKeyStatusUseCase(
	identityRepo useridentitykey.Repository,
	signedPreKeyRepo usersignedprekey.Repository,
	otpPreKeyRepo userotpprekey.Repository,
) *CheckKeyStatusUseCase {
	if otpPreKeyRepo == nil {
		otpPreKeyRepo = &checkKeyStatusOTPPreKeyRepoStub{}
	}
	return NewCheckKeyStatusUseCase(identityRepo, signedPreKeyRepo, otpPreKeyRepo)
}

func checkKeyStatusInput(deviceID string) appShared.UseCaseInput[CheckKeyStatusInput] {
	return appShared.UseCaseInput[CheckKeyStatusInput]{
		Data: CheckKeyStatusInput{
			TargetUserID: "42",
			DeviceID:     deviceID,
		},
	}
}

func TestCheckKeyStatusUseCase_BothKeysExist(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{
			findActive: func(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				return &usersignedprekey.UserSignedPreKey{}, nil
			},
		},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.IdentityKeyExists)
	assert.True(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_ReturnsPublicMaterialAndOTPCountWithoutConsuming(t *testing.T) {
	deviceIDText := "550e8400-e29b-41d4-a716-446655440000"
	deviceID, err := shared.ParseDeviceID(deviceIDText)
	require.NoError(t, err)

	var identityPub useridentitykey.PublicKey
	copy(identityPub[:], bytesOf(1, 32))
	var identitySign useridentitykey.SignPublicKey
	copy(identitySign[:], bytesOf(2, 32))
	var spkPub usersignedprekey.PublicKey
	copy(spkPub[:], bytesOf(3, 32))
	var spkSig usersignedprekey.Signature
	copy(spkSig[:], bytesOf(4, 64))

	otpRepo := &checkKeyStatusOTPPreKeyRepoStub{count: 9}
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{
					DeviceID:      deviceID,
					PublicKey:     identityPub,
					SignPublicKey: identitySign,
				}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{
			findActive: func(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				return &usersignedprekey.UserSignedPreKey{
					DeviceID:  deviceID,
					KeyID:     7,
					PublicKey: spkPub,
					Signature: spkSig,
					IsActive:  true,
				}, nil
			},
		},
		otpRepo,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceIDText))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, deviceIDText, out.DeviceID)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identityPub[:]), out.IdentityKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identitySign[:]), out.IdentityKeySign)
	assert.Equal(t, base64.StdEncoding.EncodeToString(spkPub[:]), out.SignedPreKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(spkSig[:]), out.SPKSignature)
	assert.Equal(t, uint32(7), out.SPKKeyID)
	assert.Equal(t, 9, out.OTPPreKeyCount)
	assert.Equal(t, 0, otpRepo.consumeCalls)
}

func TestCheckKeyStatusUseCase_PropagatesOTPCountError(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	repoErr := errors.New("otp count unavailable")
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{},
		&checkKeyStatusSignedPreKeyRepoStub{},
		&checkKeyStatusOTPPreKeyRepoStub{
			countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
				return 0, repoErr
			},
		},
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func bytesOf(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}

func TestCheckKeyStatusUseCase_NotFoundMapsToFalse(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{},
		&checkKeyStatusSignedPreKeyRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.IdentityKeyExists)
	assert.False(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_SignedPreKeyNotFoundMapsToFalse(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.IdentityKeyExists)
	assert.False(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_PropagatesIdentityRepoError(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	repoErr := errors.New("identity repo unavailable")
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return nil, repoErr
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_PropagatesSignedPreKeyRepoError(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	repoErr := errors.New("signed prekey repo unavailable")
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{
			findActive: func(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				return nil, repoErr
			},
		},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_PropagatesFindByUserError(t *testing.T) {
	repoErr := errors.New("identity list unavailable")
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return nil, repoErr
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(""))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_EmptyFindByUserMapsToFalse(t *testing.T) {
	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return []*useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(""))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.IdentityKeyExists)
	assert.False(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_UsesFirstIdentityKeyWhenDeviceIDMissing(t *testing.T) {
	deviceID, err := shared.ParseDeviceID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)

	uc := newCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return []*useridentitykey.UserIdentityKey{{DeviceID: deviceID}}, nil
			},
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{
			findActive: func(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				return &usersignedprekey.UserSignedPreKey{}, nil
			},
		},
		nil,
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(""))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.IdentityKeyExists)
	assert.True(t, out.SignedPreKeyExists)
}
