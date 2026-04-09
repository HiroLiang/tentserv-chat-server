package e2ee

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type handlerIdentityRepoStub struct {
	key *useridentitykey.UserIdentityKey
}

func (s handlerIdentityRepoStub) FindByUserAndDevice(context.Context, user.ID, device.ID) (*useridentitykey.UserIdentityKey, error) {
	if s.key == nil {
		return nil, useridentitykey.ErrNotFound
	}
	return s.key, nil
}

func (s handlerIdentityRepoStub) FindByUser(context.Context, user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	if s.key == nil {
		return nil, nil
	}
	return []*useridentitykey.UserIdentityKey{s.key}, nil
}

func (s handlerIdentityRepoStub) Upsert(context.Context, *useridentitykey.UserIdentityKey) error {
	return nil
}

type handlerSignedPreKeyRepoStub struct {
	key *usersignedprekey.UserSignedPreKey
}

func (s handlerSignedPreKeyRepoStub) FindActive(context.Context, user.ID, device.ID) (*usersignedprekey.UserSignedPreKey, error) {
	if s.key == nil {
		return nil, usersignedprekey.ErrNotFound
	}
	return s.key, nil
}

func (s handlerSignedPreKeyRepoStub) FindByKeyID(context.Context, user.ID, device.ID, usersignedprekey.KeyID) (*usersignedprekey.UserSignedPreKey, error) {
	return nil, usersignedprekey.ErrNotFound
}

func (s handlerSignedPreKeyRepoStub) Add(context.Context, *usersignedprekey.UserSignedPreKey) error {
	return nil
}

func (s handlerSignedPreKeyRepoStub) DeactivateAll(context.Context, user.ID, device.ID) error {
	return nil
}

type handlerOTPPreKeyRepoStub struct {
	count int
}

func (s handlerOTPPreKeyRepoStub) ConsumeOne(context.Context, user.ID, device.ID) (*userotpprekey.UserOTPPreKey, error) {
	return nil, userotpprekey.ErrPoolEmpty
}

func (s handlerOTPPreKeyRepoStub) AddBatch(context.Context, []*userotpprekey.UserOTPPreKey) error {
	return nil
}

func (s handlerOTPPreKeyRepoStub) CountAvailable(context.Context, user.ID, device.ID) (int, error) {
	return s.count, nil
}

func TestE2EEHandler_CheckKeyStatusReturnsPublicMaterial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deviceID, err := shared.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	var identityPub useridentitykey.PublicKey
	copy(identityPub[:], repeatedHandlerByte(1, 32))
	var identitySign useridentitykey.SignPublicKey
	copy(identitySign[:], repeatedHandlerByte(2, 32))
	var spkPub usersignedprekey.PublicKey
	copy(spkPub[:], repeatedHandlerByte(3, 32))
	var spkSig usersignedprekey.Signature
	copy(spkSig[:], repeatedHandlerByte(4, 64))

	handler := NewE2EEHandler(
		nil,
		nil,
		nil,
		nil,
		nil,
		usecase.NewCheckKeyStatusUseCase(
			handlerIdentityRepoStub{key: &useridentitykey.UserIdentityKey{
				DeviceID:      deviceID,
				PublicKey:     identityPub,
				SignPublicKey: identitySign,
			}},
			handlerSignedPreKeyRepoStub{key: &usersignedprekey.UserSignedPreKey{
				DeviceID:  deviceID,
				KeyID:     7,
				PublicKey: spkPub,
				Signature: spkSig,
				IsActive:  true,
			}},
			handlerOTPPreKeyRepoStub{count: 11},
		),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler.RegisterE2EERoutes(router.Group("/e2ee"))

	req := httptest.NewRequest(http.MethodGet, "/e2ee/key-status/501?device_id=11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("X-Device-ID", "11111111-1111-1111-1111-111111111111")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body CheckKeyStatusResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.IdentityKeyExists)
	assert.True(t, body.SignedPreKeyExists)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", body.DeviceID)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identityPub[:]), body.IdentityKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identitySign[:]), body.IdentityKeySign)
	assert.Equal(t, base64.StdEncoding.EncodeToString(spkPub[:]), body.SignedPreKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(spkSig[:]), body.SPKSignature)
	assert.Equal(t, uint32(7), body.SPKKeyID)
	assert.Equal(t, 11, body.OTPPreKeyCount)
}

func TestE2EEHandler_CheckKeyStatusReturnsIdentityWithoutSignedPreKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deviceID, err := shared.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	var identityPub useridentitykey.PublicKey
	copy(identityPub[:], repeatedHandlerByte(1, 32))
	var identitySign useridentitykey.SignPublicKey
	copy(identitySign[:], repeatedHandlerByte(2, 32))

	handler := NewE2EEHandler(
		nil,
		nil,
		nil,
		nil,
		nil,
		usecase.NewCheckKeyStatusUseCase(
			handlerIdentityRepoStub{key: &useridentitykey.UserIdentityKey{
				DeviceID:      deviceID,
				PublicKey:     identityPub,
				SignPublicKey: identitySign,
			}},
			handlerSignedPreKeyRepoStub{},
			handlerOTPPreKeyRepoStub{count: 0},
		),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler.RegisterE2EERoutes(router.Group("/e2ee"))

	req := httptest.NewRequest(http.MethodGet, "/e2ee/key-status/501?device_id=11111111-1111-1111-1111-111111111111", nil)
	req.Header.Set("X-Device-ID", "11111111-1111-1111-1111-111111111111")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body CheckKeyStatusResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.True(t, body.IdentityKeyExists)
	assert.False(t, body.SignedPreKeyExists)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", body.DeviceID)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identityPub[:]), body.IdentityKey)
	assert.Equal(t, base64.StdEncoding.EncodeToString(identitySign[:]), body.IdentityKeySign)
	assert.Empty(t, body.SignedPreKey)
	assert.Empty(t, body.SPKSignature)
	assert.Zero(t, body.SPKKeyID)
	assert.Equal(t, 0, body.OTPPreKeyCount)
}

func TestE2EEHandler_GetKeyPolicyReturnsConfiguredDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, config.LoadConfig("../../../../../dev-doc/config"))

	handler := NewE2EEHandler(nil, nil, nil, nil, nil, nil, usecase.NewGetKeyPolicyUseCase(), nil, nil, nil, nil)
	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler.RegisterE2EERoutes(router.Group("/e2ee"))

	req := httptest.NewRequest(http.MethodGet, "/e2ee/key-policy", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	var body GetKeyPolicyResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	assert.Equal(t, 20, body.OTPPreKeyTargetCount)
	assert.Equal(t, 5, body.OTPPreKeyReplenishThreshold)
}

func repeatedHandlerByte(value byte, length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = value
	}
	return out
}
