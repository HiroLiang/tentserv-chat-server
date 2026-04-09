package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authPort "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authTestSessionManager struct {
	findByToken func(ctx context.Context, token auth.AccessToken) (*auth.Session, error)
}

var _ authPort.SessionManager = (*authTestSessionManager)(nil)

func (s *authTestSessionManager) Create(context.Context, auth.CreateSessionInput) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *authTestSessionManager) FindByToken(ctx context.Context, token auth.AccessToken) (*auth.Session, error) {
	if s.findByToken != nil {
		return s.findByToken(ctx, token)
	}
	return nil, assert.AnError
}

func (s *authTestSessionManager) Refresh(context.Context, auth.RefreshToken) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *authTestSessionManager) Revoke(context.Context, auth.AccessToken) error {
	return nil
}

func (s *authTestSessionManager) RevokeAllForUser(context.Context, sharedDomain.AccountID) error {
	return nil
}

func (s *authTestSessionManager) RevokeAll(context.Context) error {
	return nil
}

func (s *authTestSessionManager) SwitchUser(context.Context, auth.AccessToken, sharedDomain.UserID) error {
	return nil
}

type authTestUserRepo struct{}

func (r *authTestUserRepo) Create(context.Context, *user.User) (sharedDomain.UserID, error) {
	return 0, nil
}

func (r *authTestUserRepo) FindByID(context.Context, sharedDomain.UserID) (*user.User, error) {
	return &user.User{ID: sharedDomain.UserID(123), Name: "Tester"}, nil
}

func (r *authTestUserRepo) FindByAccountID(context.Context, sharedDomain.AccountID) (*[]user.User, error) {
	return nil, nil
}

func (r *authTestUserRepo) Update(context.Context, *user.User) error {
	return nil
}

func (r *authTestUserRepo) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (r *authTestUserRepo) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (r *authTestUserRepo) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func httpAuthTestRouter(sessionManager authPort.SessionManager) *gin.Engine {
	engine := gin.New()
	engine.GET("/api/auth/profile",
		AuthMiddleware(sessionManager, &authTestUserRepo{}),
		RequireAuthMiddleware(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)
	return engine
}

func TestAuthMiddleware_WebSocketDeviceMatchCreatesAuthContext(t *testing.T) {
	logger.InitTestEnv()
	gin.SetMode(gin.TestMode)

	deviceID, err := sharedDomain.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	engine := gin.New()
	engine.GET("/ws/",
		AuthMiddleware(&authTestSessionManager{
			findByToken: func(context.Context, auth.AccessToken) (*auth.Session, error) {
				return &auth.Session{
					AccountID: sharedDomain.AccountID(1),
					UserID:    sharedDomain.UserID(123),
					DeviceID:  deviceID,
				}, nil
			},
		}, &authTestUserRepo{}),
		RequireAuthMiddleware(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/ws/?token=test-token&device_id=11111111-1111-1111-1111-111111111111", nil)
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestAuthMiddleware_WebSocketDeviceMismatchReturnsUnauthorized(t *testing.T) {
	logger.InitTestEnv()
	gin.SetMode(gin.TestMode)

	deviceID, err := sharedDomain.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	engine := gin.New()
	engine.GET("/ws/",
		AuthMiddleware(&authTestSessionManager{
			findByToken: func(context.Context, auth.AccessToken) (*auth.Session, error) {
				return &auth.Session{
					AccountID: sharedDomain.AccountID(1),
					UserID:    sharedDomain.UserID(123),
					DeviceID:  deviceID,
				}, nil
			},
		}, &authTestUserRepo{}),
		RequireAuthMiddleware(),
		func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/ws/?token=test-token&device_id=22222222-2222-2222-2222-222222222222", nil)
	resp := httptest.NewRecorder()

	engine.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthMiddleware_HTTPMalformedAuthorizationHeaderReturnsUnauthorized(t *testing.T) {
	logger.InitTestEnv()
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Token malformed")
	req.Header.Set("X-Device-ID", "11111111-1111-1111-1111-111111111111")
	resp := httptest.NewRecorder()

	httpAuthTestRouter(&authTestSessionManager{}).ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthMiddleware_HTTPTamperedBearerTokenReturnsUnauthorized(t *testing.T) {
	logger.InitTestEnv()
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer tampered-token")
	req.Header.Set("X-Device-ID", "11111111-1111-1111-1111-111111111111")
	resp := httptest.NewRecorder()

	httpAuthTestRouter(&authTestSessionManager{
		findByToken: func(context.Context, auth.AccessToken) (*auth.Session, error) {
			return nil, auth.ErrSessionNotFound
		},
	}).ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAuthMiddleware_HTTPDeviceMismatchReturnsUnauthorized(t *testing.T) {
	logger.InitTestEnv()
	gin.SetMode(gin.TestMode)

	deviceID, err := sharedDomain.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("X-Device-ID", "22222222-2222-2222-2222-222222222222")
	resp := httptest.NewRecorder()

	httpAuthTestRouter(&authTestSessionManager{
		findByToken: func(context.Context, auth.AccessToken) (*auth.Session, error) {
			return &auth.Session{
				AccountID: sharedDomain.AccountID(1),
				UserID:    sharedDomain.UserID(123),
				DeviceID:  deviceID,
			}, nil
		},
	}).ServeHTTP(resp, req)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}
