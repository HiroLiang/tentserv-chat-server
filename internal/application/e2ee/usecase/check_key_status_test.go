package usecase

import (
	"context"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
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
	uc := NewCheckKeyStatusUseCase(
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
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.IdentityKeyExists)
	assert.True(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_NotFoundMapsToFalse(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	uc := NewCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{},
		&checkKeyStatusSignedPreKeyRepoStub{},
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.False(t, out.IdentityKeyExists)
	assert.False(t, out.SignedPreKeyExists)
}

func TestCheckKeyStatusUseCase_SignedPreKeyNotFoundMapsToFalse(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	uc := NewCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return &useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
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
	uc := NewCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return nil, repoErr
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_PropagatesSignedPreKeyRepoError(t *testing.T) {
	deviceID := "550e8400-e29b-41d4-a716-446655440000"
	repoErr := errors.New("signed prekey repo unavailable")
	uc := NewCheckKeyStatusUseCase(
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
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(deviceID))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_PropagatesFindByUserError(t *testing.T) {
	repoErr := errors.New("identity list unavailable")
	uc := NewCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return nil, repoErr
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(""))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestCheckKeyStatusUseCase_EmptyFindByUserMapsToFalse(t *testing.T) {
	uc := NewCheckKeyStatusUseCase(
		&checkKeyStatusIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return []*useridentitykey.UserIdentityKey{}, nil
			},
		},
		&checkKeyStatusSignedPreKeyRepoStub{},
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

	uc := NewCheckKeyStatusUseCase(
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
	)

	out, err := uc.Execute(context.Background(), checkKeyStatusInput(""))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.True(t, out.IdentityKeyExists)
	assert.True(t, out.SignedPreKeyExists)
}
