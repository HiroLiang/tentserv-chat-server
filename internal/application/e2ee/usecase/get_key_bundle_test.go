package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/deliveryqueue"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keyBundleIdentityRepoStub struct {
	findByUserAndDevice func(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error)
	findByUser          func(ctx context.Context, userID user.ID) ([]*useridentitykey.UserIdentityKey, error)
}

func (s *keyBundleIdentityRepoStub) FindByUserAndDevice(ctx context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
	if s.findByUserAndDevice != nil {
		return s.findByUserAndDevice(ctx, userID, deviceID)
	}
	return nil, useridentitykey.ErrNotFound
}

func (s *keyBundleIdentityRepoStub) FindByUser(ctx context.Context, userID user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	if s.findByUser != nil {
		return s.findByUser(ctx, userID)
	}
	return nil, nil
}

func (s *keyBundleIdentityRepoStub) Upsert(context.Context, *useridentitykey.UserIdentityKey) error {
	return nil
}

type keyBundleSignedPreKeyRepoStub struct {
	findActive func(ctx context.Context, userID user.ID, deviceID device.ID) (*usersignedprekey.UserSignedPreKey, error)
}

func (s *keyBundleSignedPreKeyRepoStub) FindActive(ctx context.Context, userID user.ID, deviceID device.ID) (*usersignedprekey.UserSignedPreKey, error) {
	if s.findActive != nil {
		return s.findActive(ctx, userID, deviceID)
	}
	return nil, usersignedprekey.ErrNotFound
}

func (s *keyBundleSignedPreKeyRepoStub) FindByKeyID(context.Context, user.ID, device.ID, usersignedprekey.KeyID) (*usersignedprekey.UserSignedPreKey, error) {
	return nil, usersignedprekey.ErrNotFound
}

func (s *keyBundleSignedPreKeyRepoStub) Add(context.Context, *usersignedprekey.UserSignedPreKey) error {
	return nil
}

func (s *keyBundleSignedPreKeyRepoStub) DeactivateAll(context.Context, user.ID, device.ID) error {
	return nil
}

type keyBundleOTPPreKeyRepoStub struct {
	consumeOne     func(ctx context.Context, userID user.ID, deviceID device.ID) (*userotpprekey.UserOTPPreKey, error)
	countAvailable func(ctx context.Context, userID user.ID, deviceID device.ID) (int, error)
	consumeCalls   int
}

func (s *keyBundleOTPPreKeyRepoStub) ConsumeOne(ctx context.Context, userID user.ID, deviceID device.ID) (*userotpprekey.UserOTPPreKey, error) {
	s.consumeCalls++
	if s.consumeOne != nil {
		return s.consumeOne(ctx, userID, deviceID)
	}
	return nil, userotpprekey.ErrPoolEmpty
}

func (s *keyBundleOTPPreKeyRepoStub) AddBatch(context.Context, []*userotpprekey.UserOTPPreKey) error {
	return nil
}

func (s *keyBundleOTPPreKeyRepoStub) CountAvailable(ctx context.Context, userID user.ID, deviceID device.ID) (int, error) {
	if s.countAvailable != nil {
		return s.countAvailable(ctx, userID, deviceID)
	}
	return 0, nil
}

type keyBundleDispatch struct {
	userID  shared.UserID
	msgType string
	payload []byte
	calls   int
}

func (d *keyBundleDispatch) Dispatch(_ context.Context, userID shared.UserID, msgType string, payload []byte) error {
	d.calls++
	d.userID = userID
	d.msgType = msgType
	d.payload = payload
	return nil
}

func TestGetKeyBundleUseCase_ConsumesOTPAndReturnsBundle(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	otpRepo := &keyBundleOTPPreKeyRepoStub{
		consumeOne: func(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
			return material.otp, nil
		},
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 10, nil
		},
	}
	uc := newKeyBundleUseCaseForTest(t, material, otpRepo, nil, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
	require.NotNil(t, out.OTPPreKey)
	require.NotNil(t, out.OTPPreKeyID)
	assert.Equal(t, base64.StdEncoding.EncodeToString(material.identity.PublicKey[:]), out.IdentityKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(material.identity.SignPublicKey[:]), out.IdentityKeySign)
	assert.Equal(t, base64.StdEncoding.EncodeToString(material.spk.PublicKey[:]), out.SignedPreKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(material.spk.Signature[:]), out.SPKSignature)
	assert.Equal(t, uint32(material.spk.KeyID), out.SPKKeyID)
	assert.Equal(t, base64.StdEncoding.EncodeToString(material.otp.PublicKey[:]), *out.OTPPreKey)
	assert.Equal(t, uint32(material.otp.KeyID), *out.OTPPreKeyID)
	assert.Equal(t, 1, otpRepo.consumeCalls)
}

func TestGetKeyBundleUseCase_PoolEmptyFallsBackToIdentityAndSPK(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{}, nil, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Nil(t, out.OTPPreKey)
	assert.Nil(t, out.OTPPreKeyID)
	assert.NotEmpty(t, out.IdentityKey)
	assert.NotEmpty(t, out.SignedPreKey)
}

func TestGetKeyBundleUseCase_UnexpectedConsumeErrorReturnsError(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	repoErr := errors.New("database unavailable")
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{
		consumeOne: func(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
			return nil, repoErr
		},
	}, nil, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, repoErr)
}

func TestGetKeyBundleUseCase_IdentityMissingReturnsNotFound(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	uc := newGetKeyBundleUseCase(
		&keyBundleIdentityRepoStub{},
		&keyBundleSignedPreKeyRepoStub{},
		&keyBundleOTPPreKeyRepoStub{},
		nil,
		keyBundlePolicy(20, 5),
	)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrIdentityNotFound)
}

func TestGetKeyBundleUseCase_SPKMissingReturnsKeyBundleNotFound(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	uc := newGetKeyBundleUseCase(
		&keyBundleIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return material.identity, nil
			},
		},
		&keyBundleSignedPreKeyRepoStub{},
		&keyBundleOTPPreKeyRepoStub{},
		nil,
		keyBundlePolicy(20, 5),
	)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.Error(t, err)
	assert.Nil(t, out)
	assert.ErrorIs(t, err, ErrKeyBundleNotFound)
}

func TestGetKeyBundleUseCase_FallsBackToFirstIdentityDeviceWhenDeviceIDMissing(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	uc := newGetKeyBundleUseCase(
		&keyBundleIdentityRepoStub{
			findByUser: func(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
				return []*useridentitykey.UserIdentityKey{material.identity}, nil
			},
			findByUserAndDevice: func(_ context.Context, _ user.ID, gotDeviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
				require.Equal(t, deviceID, gotDeviceID)
				return material.identity, nil
			},
		},
		&keyBundleSignedPreKeyRepoStub{
			findActive: func(_ context.Context, _ user.ID, gotDeviceID device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				require.Equal(t, deviceID, gotDeviceID)
				return material.spk, nil
			},
		},
		&keyBundleOTPPreKeyRepoStub{},
		nil,
		keyBundlePolicy(20, 5),
	)

	out, err := uc.Execute(context.Background(), keyBundleInput(""))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotEmpty(t, out.IdentityKey)
}

func TestGetKeyBundleUseCase_DispatchesReplenishEventWhenBelowThreshold(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	dispatcher := &keyBundleDispatch{}
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{
		consumeOne: func(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
			return material.otp, nil
		},
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 4, nil
		},
	}, dispatcher, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, dispatcher.calls)
	assert.Equal(t, shared.UserID(42), dispatcher.userID)
	assert.Equal(t, string(deliveryqueue.PayloadTypeReplenishOTP), dispatcher.msgType)
	var payload struct {
		UserID   int64  `json:"user_id"`
		DeviceID string `json:"device_id"`
	}
	require.NoError(t, json.Unmarshal(dispatcher.payload, &payload))
	assert.Equal(t, int64(42), payload.UserID)
	assert.Equal(t, deviceID.String(), payload.DeviceID)
}

func TestGetKeyBundleUseCase_DoesNotDispatchAtThreshold(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	dispatcher := &keyBundleDispatch{}
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 5, nil
		},
	}, dispatcher, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Zero(t, dispatcher.calls)
}

func TestGetKeyBundleUseCase_CountErrorKeepsBundleAndSkipsDispatch(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	dispatcher := &keyBundleDispatch{}
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{
		consumeOne: func(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
			return material.otp, nil
		},
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 0, errors.New("count unavailable")
		},
	}, dispatcher, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.NotNil(t, out.OTPPreKey)
	assert.Zero(t, dispatcher.calls)
}

func TestGetKeyBundleUseCase_NilDispatcherDoesNotPanic(t *testing.T) {
	deviceID := mustKeyBundleDeviceID(t)
	material := newKeyBundleMaterial(deviceID)
	uc := newKeyBundleUseCaseForTest(t, material, &keyBundleOTPPreKeyRepoStub{
		countAvailable: func(context.Context, user.ID, device.ID) (int, error) {
			return 0, nil
		},
	}, nil, 20, 5)

	out, err := uc.Execute(context.Background(), keyBundleInput(deviceID.String()))

	require.NoError(t, err)
	require.NotNil(t, out)
}

type keyBundleMaterial struct {
	identity *useridentitykey.UserIdentityKey
	spk      *usersignedprekey.UserSignedPreKey
	otp      *userotpprekey.UserOTPPreKey
}

func newKeyBundleMaterial(deviceID device.ID) keyBundleMaterial {
	var identityPub useridentitykey.PublicKey
	copy(identityPub[:], keyBundleBytesOf(1, 32))
	var identitySign useridentitykey.SignPublicKey
	copy(identitySign[:], keyBundleBytesOf(2, 32))
	var spkPub usersignedprekey.PublicKey
	copy(spkPub[:], keyBundleBytesOf(3, 32))
	var spkSig usersignedprekey.Signature
	copy(spkSig[:], keyBundleBytesOf(4, 64))
	var otpPub userotpprekey.PublicKey
	copy(otpPub[:], keyBundleBytesOf(5, 32))

	return keyBundleMaterial{
		identity: &useridentitykey.UserIdentityKey{
			UserID:        42,
			DeviceID:      deviceID,
			PublicKey:     identityPub,
			SignPublicKey: identitySign,
		},
		spk: &usersignedprekey.UserSignedPreKey{
			UserID:    42,
			DeviceID:  deviceID,
			KeyID:     7,
			PublicKey: spkPub,
			Signature: spkSig,
			IsActive:  true,
		},
		otp: &userotpprekey.UserOTPPreKey{
			UserID:    42,
			DeviceID:  deviceID,
			KeyID:     99,
			PublicKey: otpPub,
		},
	}
}

func newKeyBundleUseCaseForTest(
	t *testing.T,
	material keyBundleMaterial,
	otpRepo *keyBundleOTPPreKeyRepoStub,
	dispatcher *keyBundleDispatch,
	target int,
	threshold int,
) *GetKeyBundleUseCase {
	t.Helper()
	var dispatch appPush.Dispatcher
	if dispatcher != nil {
		dispatch = dispatcher
	}
	return newGetKeyBundleUseCase(
		&keyBundleIdentityRepoStub{
			findByUserAndDevice: func(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
				return material.identity, nil
			},
		},
		&keyBundleSignedPreKeyRepoStub{
			findActive: func(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
				return material.spk, nil
			},
		},
		otpRepo,
		dispatch,
		keyBundlePolicy(target, threshold),
	)
}

func keyBundleInput(deviceID string) appShared.UseCaseInput[GetKeyBundleInput] {
	return appShared.UseCaseInput[GetKeyBundleInput]{
		Data: GetKeyBundleInput{
			TargetUserID: "42",
			DeviceID:     deviceID,
		},
	}
}

func keyBundlePolicy(target int, threshold int) func() keyPolicyConfig {
	return func() keyPolicyConfig {
		return keyPolicyConfig{
			OTPPreKeyTargetCount:        target,
			OTPPreKeyReplenishThreshold: threshold,
		}
	}
}

func mustKeyBundleDeviceID(t *testing.T) device.ID {
	t.Helper()
	deviceID, err := shared.ParseDeviceID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	return deviceID
}

func keyBundleBytesOf(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}
