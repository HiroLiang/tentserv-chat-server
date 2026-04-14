package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	authPort "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

const verificationExpiryTolerance = 5 * time.Second

func TestMain(m *testing.M) {
	if err := config.LoadConfig("../../../../../dev-doc/config"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type accountHandlerTxStub struct {
	commitCalls int
}

func (s *accountHandlerTxStub) Commit() error {
	s.commitCalls++
	return nil
}

func (s *accountHandlerTxStub) Rollback() error {
	return nil
}

type accountHandlerUOWStub struct {
	tx *accountHandlerTxStub
}

func (s *accountHandlerUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	if s.tx == nil {
		s.tx = &accountHandlerTxStub{}
	}
	return ctx, s.tx, nil
}

type accountHandlerHasherStub struct {
	err          error
	verifyResult bool
}

func (s accountHandlerHasherStub) Hash(string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "argon2-hash", nil
}

func (s accountHandlerHasherStub) HashBytes(bytes []byte) (string, error) {
	return s.Hash(string(bytes))
}

func (s accountHandlerHasherStub) Verify(string, string) bool {
	return s.verifyResult
}

type accountHandlerAccountRepoStub struct {
	emailExists   bool
	accountExists bool
	createErr     error
	loginPassword string
	loginStatus   domainaccount.Status
	loginUserIDs  []shared.UserID
}

func (s accountHandlerAccountRepoStub) FindByID(context.Context, shared.AccountID) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (s accountHandlerAccountRepoStub) FindByAccountName(_ context.Context, name string) (*domainaccount.Account, error) {
	if s.accountExists {
		status := s.loginStatus
		if status == "" {
			status = domainaccount.Active
		}
		return &domainaccount.Account{ID: 1, AccountName: name, Password: s.loginPassword, Status: status, UserIDs: append([]shared.UserID(nil), s.loginUserIDs...)}, nil
	}
	return nil, domainaccount.ErrAccountNotFound
}

func (s accountHandlerAccountRepoStub) FindByEmail(_ context.Context, email shared.EmailAddress) (*domainaccount.Account, error) {
	if s.emailExists {
		status := s.loginStatus
		if status == "" {
			status = domainaccount.Active
		}
		return &domainaccount.Account{ID: 1, Email: email, AccountName: "login_account", Password: s.loginPassword, Status: status, UserIDs: append([]shared.UserID(nil), s.loginUserIDs...)}, nil
	}
	return nil, domainaccount.ErrAccountNotFound
}

func (s accountHandlerAccountRepoStub) Create(context.Context, *domainaccount.Account) (shared.AccountID, error) {
	if s.createErr != nil {
		return 0, s.createErr
	}
	return 101, nil
}

func (s accountHandlerAccountRepoStub) Update(context.Context, *domainaccount.Account) error {
	return nil
}

func (s accountHandlerAccountRepoStub) RegisterDevice(context.Context, *domainaccount.AccountDevice) error {
	return nil
}

func (s accountHandlerAccountRepoStub) UpdateDeviceStatus(context.Context, shared.AccountID, shared.DeviceID, domainaccount.DeviceStatus) error {
	return nil
}

func (s accountHandlerAccountRepoStub) RecordLoginEvent(context.Context, *domainaccount.AccountLoginEvent) error {
	return nil
}

func (s accountHandlerAccountRepoStub) ReplaceDevices(context.Context, shared.AccountID, []domainaccount.AccountDevice) error {
	return nil
}

type accountHandlerUserRepoStub struct{}

func (accountHandlerUserRepoStub) Create(_ context.Context, u *domainuser.User) (shared.UserID, error) {
	u.ID = 501
	return 501, nil
}

func (accountHandlerUserRepoStub) FindByID(context.Context, shared.UserID) (*domainuser.User, error) {
	return nil, domainuser.ErrUserNotFound
}

func (accountHandlerUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (accountHandlerUserRepoStub) Update(context.Context, *domainuser.User) error {
	return nil
}

func (accountHandlerUserRepoStub) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (accountHandlerUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (accountHandlerUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

type accountHandlerUserRoleRepoStub struct{}

func (accountHandlerUserRoleRepoStub) FindRolesByUser(context.Context, shared.UserID) ([]*role.Role, error) {
	return nil, nil
}

func (accountHandlerUserRoleRepoStub) Exists(context.Context, shared.UserID, role.Code) bool {
	return false
}

func (accountHandlerUserRoleRepoStub) Assign(context.Context, shared.UserID, role.Code) error {
	return nil
}

func (accountHandlerUserRoleRepoStub) Revoke(context.Context, shared.UserID, role.Code) error {
	return nil
}

type accountHandlerSessionManagerStub struct{}

func (accountHandlerSessionManagerStub) Create(context.Context, auth.CreateSessionInput) (auth.TokenPair, error) {
	return auth.TokenPair{
		AccessToken:  auth.AccessToken("handler-access-token"),
		RefreshToken: auth.RefreshToken("handler-refresh-token"),
		ExpiresAt:    time.Now().Add(time.Hour),
	}, nil
}

func (accountHandlerSessionManagerStub) FindByToken(context.Context, auth.AccessToken) (*auth.Session, error) {
	return nil, auth.ErrSessionNotFound
}

func (accountHandlerSessionManagerStub) Refresh(context.Context, auth.RefreshToken) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (accountHandlerSessionManagerStub) Revoke(context.Context, auth.AccessToken) error {
	return nil
}

func (accountHandlerSessionManagerStub) RevokeAllForUser(context.Context, shared.AccountID) error {
	return nil
}

func (accountHandlerSessionManagerStub) RevokeAll(context.Context) error {
	return nil
}

func (accountHandlerSessionManagerStub) SwitchUser(context.Context, auth.AccessToken, shared.UserID) error {
	return nil
}

type accountHandlerDeviceRepoStub struct {
	deviceID shared.DeviceID
}

func (s accountHandlerDeviceRepoStub) FindByID(_ context.Context, id shared.DeviceID) (*domaindevice.Device, error) {
	if s.deviceID.String() != id.String() {
		return nil, domaindevice.ErrDeviceNotFound
	}
	return &domaindevice.Device{ID: id, Platform: domaindevice.MacOS, Name: "Handler Mac"}, nil
}

func (s accountHandlerDeviceRepoStub) FindAllByAccountID(context.Context, shared.AccountID) ([]*domaindevice.Device, error) {
	return nil, nil
}

func (s accountHandlerDeviceRepoStub) Create(context.Context, *domaindevice.Device) error {
	return nil
}

func (s accountHandlerDeviceRepoStub) Update(context.Context, *domaindevice.Device) error {
	return nil
}

func (s accountHandlerDeviceRepoStub) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (s accountHandlerDeviceRepoStub) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

type accountHandlerParticipantRepoStub struct{}

func (accountHandlerParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (accountHandlerParticipantRepoStub) FindByUserID(context.Context, shared.UserID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (accountHandlerParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (accountHandlerParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (accountHandlerParticipantRepoStub) Create(_ context.Context, p *participant.Participant) error {
	p.ID = 801
	p.CreatedAt = time.Now()
	return nil
}

type accountHandlerVerificationStoreStub struct{}

func (accountHandlerVerificationStoreStub) Store(context.Context, string, authPort.VerificationSession, time.Duration) error {
	return nil
}

func (accountHandlerVerificationStoreStub) Get(context.Context, string) (authPort.VerificationSession, bool, error) {
	return authPort.VerificationSession{}, false, nil
}

func (accountHandlerVerificationStoreStub) FindTokenByAccountID(context.Context, int64) (string, bool, error) {
	return "", false, nil
}

func (accountHandlerVerificationStoreStub) Delete(context.Context, string) error {
	return nil
}

type accountHandlerEmailServiceStub struct{}

func (accountHandlerEmailServiceStub) Send(context.Context, appEmail.EmailBuilder) error {
	return nil
}

type accountHandlerEmailBuilderStub struct{}

func (accountHandlerEmailBuilderStub) BuildEmail(context.Context) (*shared.Email, error) {
	return &shared.Email{}, nil
}

type accountHandlerVerifyEmailStoreStub struct {
	sessions      map[string]authPort.VerificationSession
	accountTokens map[int64]string
	getCalls      int
	deleteCalls   int
}

func newAccountHandlerVerifyEmailStoreStub() *accountHandlerVerifyEmailStoreStub {
	return &accountHandlerVerifyEmailStoreStub{
		sessions:      map[string]authPort.VerificationSession{},
		accountTokens: map[int64]string{},
	}
}

func (s *accountHandlerVerifyEmailStoreStub) Store(_ context.Context, token string, session authPort.VerificationSession, _ time.Duration) error {
	s.sessions[token] = session
	s.accountTokens[session.AccountID] = token
	return nil
}

func (s *accountHandlerVerifyEmailStoreStub) Get(_ context.Context, token string) (authPort.VerificationSession, bool, error) {
	s.getCalls++
	session, ok := s.sessions[token]
	return session, ok, nil
}

func (s *accountHandlerVerifyEmailStoreStub) FindTokenByAccountID(_ context.Context, accountID int64) (string, bool, error) {
	token, ok := s.accountTokens[accountID]
	return token, ok, nil
}

func (s *accountHandlerVerifyEmailStoreStub) Delete(_ context.Context, token string) error {
	s.deleteCalls++
	session, ok := s.sessions[token]
	if ok {
		delete(s.accountTokens, session.AccountID)
	}
	delete(s.sessions, token)
	return nil
}

type accountHandlerVerifyEmailAccountRepoStub struct {
	accountsByID map[shared.AccountID]*domainaccount.Account
	updateCalls  int
	lastUpdated  *domainaccount.Account
}

func newAccountHandlerVerifyEmailAccountRepoStub() *accountHandlerVerifyEmailAccountRepoStub {
	return &accountHandlerVerifyEmailAccountRepoStub{
		accountsByID: map[shared.AccountID]*domainaccount.Account{},
	}
}

func (s *accountHandlerVerifyEmailAccountRepoStub) FindByID(_ context.Context, id shared.AccountID) (*domainaccount.Account, error) {
	acc, ok := s.accountsByID[id]
	if !ok {
		return nil, domainaccount.ErrAccountNotFound
	}
	cloned := *acc
	return &cloned, nil
}

func (s *accountHandlerVerifyEmailAccountRepoStub) FindByAccountName(context.Context, string) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (s *accountHandlerVerifyEmailAccountRepoStub) FindByEmail(context.Context, shared.EmailAddress) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (s *accountHandlerVerifyEmailAccountRepoStub) Create(context.Context, *domainaccount.Account) (shared.AccountID, error) {
	return 0, domainaccount.ErrAccountExist
}

func (s *accountHandlerVerifyEmailAccountRepoStub) Update(_ context.Context, acc *domainaccount.Account) error {
	s.updateCalls++
	cloned := *acc
	s.accountsByID[cloned.ID] = &cloned
	s.lastUpdated = &cloned
	return nil
}

func (s *accountHandlerVerifyEmailAccountRepoStub) RegisterDevice(context.Context, *domainaccount.AccountDevice) error {
	return nil
}

func (s *accountHandlerVerifyEmailAccountRepoStub) UpdateDeviceStatus(context.Context, shared.AccountID, shared.DeviceID, domainaccount.DeviceStatus) error {
	return nil
}

func (s *accountHandlerVerifyEmailAccountRepoStub) RecordLoginEvent(context.Context, *domainaccount.AccountLoginEvent) error {
	return nil
}

func (s *accountHandlerVerifyEmailAccountRepoStub) ReplaceDevices(context.Context, shared.AccountID, []domainaccount.AccountDevice) error {
	return nil
}

func newAccountHandlerRouter(emailExists, accountExists bool, createErr error) (*gin.Engine, *accountHandlerUOWStub) {
	gin.SetMode(gin.TestMode)
	uow := &accountHandlerUOWStub{}
	registerUseCase := authUseCase.NewRegisterUseCase(
		uow,
		accountHandlerHasherStub{},
		accountHandlerAccountRepoStub{emailExists: emailExists, accountExists: accountExists, createErr: createErr},
		accountHandlerUserRepoStub{},
		accountHandlerUserRoleRepoStub{},
		accountHandlerVerificationStoreStub{},
		accountHandlerEmailServiceStub{},
		func(string, string, string) appEmail.EmailBuilder {
			return accountHandlerEmailBuilderStub{}
		},
	)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler := NewAuthHandler(registerUseCase, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.RegisterAuthRoutes(router.Group("/api/auth"))
	return router, uow
}

func newAccountHandlerVerifyEmailRouter(
	store *accountHandlerVerifyEmailStoreStub,
	accountRepo *accountHandlerVerifyEmailAccountRepoStub,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	verifyEmailUseCase := authUseCase.NewVerifyEmailUseCase(store, accountRepo)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler := NewAuthHandler(nil, nil, nil, nil, verifyEmailUseCase, nil, nil, nil, nil)
	handler.RegisterAuthRoutes(router.Group("/api/auth"))
	return router
}

func newAccountHandlerResendVerifyEmailRouter(
	store *accountHandlerVerifyEmailStoreStub,
	accountRepo *accountHandlerVerifyEmailAccountRepoStub,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	resendVerifyEmailUseCase := authUseCase.NewResendVerifyEmailUseCase(
		store,
		accountRepo,
		accountHandlerEmailServiceStub{},
		func(string, string, string) appEmail.EmailBuilder {
			return accountHandlerEmailBuilderStub{}
		},
	)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler := NewAuthHandler(nil, nil, nil, nil, nil, resendVerifyEmailUseCase, nil, nil, nil)
	handler.RegisterAuthRoutes(router.Group("/api/auth"))
	return router
}

func newAccountHandlerLoginRouter() (*gin.Engine, *accountHandlerUOWStub) {
	gin.SetMode(gin.TestMode)
	uow := &accountHandlerUOWStub{}
	deviceID, _ := shared.ParseDeviceID("11111111-1111-1111-1111-111111111111")
	loginUseCase := authUseCase.NewLoginUseCase(
		uow,
		accountHandlerHasherStub{verifyResult: true},
		nil,
		accountHandlerSessionManagerStub{},
		nil,
		accountHandlerAccountRepoStub{
			emailExists:   true,
			loginPassword: "stored-hash",
		},
		accountHandlerUserRepoStub{},
		accountHandlerUserRoleRepoStub{},
		accountHandlerDeviceRepoStub{deviceID: deviceID},
		accountHandlerParticipantRepoStub{},
		nil,
		accountHandlerEmailServiceStub{},
		func(string, string, string, string, string, time.Time) appEmail.EmailBuilder {
			return accountHandlerEmailBuilderStub{}
		},
		func(string, string, string, string, string, string, time.Time) appEmail.EmailBuilder {
			return accountHandlerEmailBuilderStub{}
		},
	)

	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler := NewAuthHandler(nil, loginUseCase, nil, nil, nil, nil, nil, nil, nil)
	handler.RegisterAuthRoutes(router.Group("/api/auth"))
	return router, uow
}

func newAccountHandlerLoginInvalidPayloadRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ContextMiddleware())
	handler := NewAuthHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.RegisterAuthRoutes(router.Group("/api/auth"))
	return router
}

func performAccountRegisterRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performAccountLoginRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TentservDesktop/HandlerTest")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performAccountVerifyEmailRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performAccountResendVerifyEmailRequest(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/resend-verify-email", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func decodeAccountHandlerError(t *testing.T, body *bytes.Buffer) response.ErrorResponse {
	t.Helper()
	var errResp response.ErrorResponse
	if err := json.Unmarshal(body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v; body=%s", err, body.String())
	}
	return errResp
}

func assertVerificationExpiryContract(t *testing.T, expiresAtMS int64, startedAt, completedAt time.Time) {
	t.Helper()

	if expiresAtMS <= completedAt.UnixMilli() {
		t.Fatalf("expected future verification expiry in epoch milliseconds, got expires_at_ms=%d now_ms=%d", expiresAtMS, completedAt.UnixMilli())
	}

	ttl := config.App().Email.VerifyTTL
	minExpected := startedAt.Add(ttl - verificationExpiryTolerance).UnixMilli()
	maxExpected := completedAt.Add(ttl + verificationExpiryTolerance).UnixMilli()
	if expiresAtMS < minExpected || expiresAtMS > maxExpected {
		t.Fatalf(
			"expected verification expiry to match configured ttl=%s within tolerance=%s, got expires_at_ms=%d expected_range=[%d,%d]",
			ttl,
			verificationExpiryTolerance,
			expiresAtMS,
			minExpected,
			maxExpected,
		)
	}
}

func TestAuthHandler_RegisterSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	router, uow := newAccountHandlerRouter(false, false, nil)
	body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`

	t.Log("Given: register usecase dependencies succeed")
	t.Log("Input: valid register JSON with email/account/display_name/password_present=true")
	t.Log("Action: POST /api/auth/register")

	resp := performAccountRegisterRequest(router, body)

	t.Logf("Output: status=%d body=%s", resp.Code, resp.Body.String())
	t.Logf("Mutation: commit_calls=%d", uow.tx.commitCalls)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}
	var out RegisterResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode register response: %v body=%s", err, resp.Body.String())
	}
	if out.VerificationToken == "" {
		t.Fatalf("expected verification token, got %+v", out)
	}
	assertVerificationExpiryContract(t, out.VerificationExpiresAtMS, start, time.Now())
}

func TestAuthHandler_ResendVerifyEmailSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	store.sessions["active-token"] = authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(3 * time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}
	store.accountTokens[101] = "active-token"
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	accountRepo.accountsByID[101] = &domainaccount.Account{
		ID:          101,
		Email:       shared.EmailAddress("applying@example.com"),
		AccountName: "applying_account",
		Status:      domainaccount.Applying,
	}
	router := newAccountHandlerResendVerifyEmailRouter(store, accountRepo)

	t.Log("Given: resend verify email token maps to an applying account")
	t.Log("Input: token_present=true account_status=Applying")
	t.Log("Action: POST /api/auth/resend-verify-email")

	resp := performAccountResendVerifyEmailRequest(router, `{"token":"active-token"}`)
	body := resp.Body.String()

	t.Logf("Output: status=%d body=%s", resp.Code, body)
	t.Logf("Mutation: get_calls=%d delete_calls=%d sessions_in_store=%d", store.getCalls, store.deleteCalls, len(store.sessions))
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, body)
	}

	var out ResendVerifyEmailResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode resend verify email response: %v body=%s", err, resp.Body.String())
	}
	if out.VerificationToken == "" || out.VerificationToken == "active-token" {
		t.Fatalf("expected a fresh verification token, got %+v", out)
	}
	assertVerificationExpiryContract(t, out.VerificationExpiresAtMS, start, time.Now())

	session, ok := store.sessions[out.VerificationToken]
	if !ok {
		t.Fatalf("expected returned token to remain stored, got token=%q store=%v", out.VerificationToken, store.sessions)
	}
	if session.ExpiresAtMS != out.VerificationExpiresAtMS {
		t.Fatalf("expected response expiry to match stored session expiry, got response=%d store=%d", out.VerificationExpiresAtMS, session.ExpiresAtMS)
	}
	if store.deleteCalls != 1 {
		t.Fatalf("expected old token to be deleted once, got delete_calls=%d", store.deleteCalls)
	}
}

func TestAuthHandler_ResendVerifyEmailExpiredRetainedTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	store.sessions["expired-token"] = authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(-time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}
	store.accountTokens[101] = "expired-token"
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	accountRepo.accountsByID[101] = &domainaccount.Account{
		ID:          101,
		Email:       shared.EmailAddress("applying@example.com"),
		AccountName: "applying_account",
		Status:      domainaccount.Applying,
	}
	router := newAccountHandlerResendVerifyEmailRouter(store, accountRepo)

	t.Log("Given: resend verify email token is expired by expires_at_ms but still retained in the store")
	t.Log("Input: token_present=true token_expired=true account_status=Applying")
	t.Log("Action: POST /api/auth/resend-verify-email")

	resp := performAccountResendVerifyEmailRequest(router, `{"token":"expired-token"}`)
	body := resp.Body.String()

	t.Logf("Output: status=%d body=%s", resp.Code, body)
	t.Logf("Mutation: get_calls=%d delete_calls=%d sessions_in_store=%d", store.getCalls, store.deleteCalls, len(store.sessions))
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, body)
	}

	var out ResendVerifyEmailResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode resend verify email response: %v body=%s", err, resp.Body.String())
	}
	if out.VerificationToken == "" || out.VerificationToken == "expired-token" {
		t.Fatalf("expected a fresh verification token, got %+v", out)
	}
	assertVerificationExpiryContract(t, out.VerificationExpiresAtMS, start, time.Now())
}

func TestAuthHandler_RegisterInvalidPayloadHasStructuredLog(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "invalid email", body: `{"email":"not-an-email","account":"new_account","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`},
		{name: "missing fields", body: `{"email":"new@example.com"}`},
		{name: "password too short", body: `{"email":"new@example.com","account":"new_account","name":"New Display","password":"abc","confirm_password":"abc"}`},
		{name: "empty password", body: `{"email":"new@example.com","account":"new_account","name":"New Display","password":"","confirm_password":""}`},
		{name: "account name too long", body: `{"email":"new@example.com","account":"` + strings.Repeat("a", 51) + `","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`},
		{name: "display name too long", body: `{"email":"new@example.com","account":"new_account","name":"` + strings.Repeat("a", 101) + `","password":"redacted-password","confirm_password":"redacted-password"}`},
		{name: "email too long", body: `{"email":"` + strings.Repeat("a", 243) + `@example.com","account":"new_account","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			router, _ := newAccountHandlerRouter(false, false, nil)

			t.Log("Given: invalid register request payload")
			t.Logf("Input: case=%s body_len=%d", tc.name, len(tc.body))
			t.Log("Action: POST /api/auth/register")

			resp := performAccountRegisterRequest(router, tc.body)
			errResp := decodeAccountHandlerError(t, resp.Body)

			t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
			t.Log("Mutation: register usecase not invoked")
			t.Logf("Duration: %s", time.Since(start))

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
			}
			if errResp.Code != "INVALID_REQUEST" {
				t.Fatalf("expected INVALID_REQUEST, got %+v", errResp)
			}
		})
	}
}

func TestAuthHandler_RegisterConflictHasStructuredLog(t *testing.T) {
	cases := []struct {
		name          string
		emailExists   bool
		accountExists bool
		wantCode      string
	}{
		{name: "email exists", emailExists: true, wantCode: "EMAIL_EXIST"},
		{name: "account exists", accountExists: true, wantCode: "ACCOUNT_EXIST"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			router, uow := newAccountHandlerRouter(tc.emailExists, tc.accountExists, nil)
			body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`

			t.Log("Given: register usecase detects an existing account identity")
			t.Logf("Input: case=%s email_exists=%t account_exists=%t", tc.name, tc.emailExists, tc.accountExists)
			t.Log("Action: POST /api/auth/register")

			resp := performAccountRegisterRequest(router, body)
			errResp := decodeAccountHandlerError(t, resp.Body)

			commitCalls := 0
			if uow.tx != nil {
				commitCalls = uow.tx.commitCalls
			}
			t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
			t.Logf("Mutation: commit_calls=%d", commitCalls)
			t.Logf("Duration: %s", time.Since(start))

			if resp.Code != http.StatusConflict {
				t.Fatalf("expected 409, got %d body=%s", resp.Code, resp.Body.String())
			}
			if errResp.Code != tc.wantCode {
				t.Fatalf("expected %s, got %+v", tc.wantCode, errResp)
			}
		})
	}
}

func TestAuthHandler_RegisterFailedHasStructuredLog(t *testing.T) {
	start := time.Now()
	router, uow := newAccountHandlerRouter(false, false, errors.New("database unavailable"))
	body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`

	t.Log("Given: register usecase dependency returns a non-conflict registration failure")
	t.Log("Input: valid register JSON with password_present=true")
	t.Log("Action: POST /api/auth/register")

	resp := performAccountRegisterRequest(router, body)
	errResp := decodeAccountHandlerError(t, resp.Body)

	commitCalls := 0
	if uow.tx != nil {
		commitCalls = uow.tx.commitCalls
	}
	t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
	t.Logf("Mutation: commit_calls=%d", commitCalls)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from register failure, got %d body=%s", resp.Code, resp.Body.String())
	}
	if errResp.Code != "REGISTER_FAILED" {
		t.Fatalf("expected REGISTER_FAILED, got %+v", errResp)
	}
}

func TestAuthHandler_RegisterUseCaseValidationHasStructuredLog(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode string
		wantHTTP int
	}{
		{
			name:     "invalid account name format",
			body:     `{"email":"new@example.com","account":"invalid account!","name":"New Display","password":"redacted-password","confirm_password":"redacted-password"}`,
			wantCode: "INVALID_ACCOUNT",
			wantHTTP: http.StatusBadRequest,
		},
		{
			name:     "common password",
			body:     `{"email":"new@example.com","account":"new_account","name":"New Display","password":"password","confirm_password":"password"}`,
			wantCode: "WEAK_PASSWORD",
			wantHTTP: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			router, _ := newAccountHandlerRouter(false, false, nil)

			t.Log("Given: register payload passes DTO binding but fails usecase validation")
			t.Logf("Input: case=%s body_len=%d", tc.name, len(tc.body))
			t.Log("Action: POST /api/auth/register")

			resp := performAccountRegisterRequest(router, tc.body)
			errResp := decodeAccountHandlerError(t, resp.Body)

			t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
			t.Log("Mutation: account not created")
			t.Logf("Duration: %s", time.Since(start))

			if resp.Code != tc.wantHTTP {
				t.Fatalf("expected %d, got %d body=%s", tc.wantHTTP, resp.Code, resp.Body.String())
			}
			if errResp.Code != tc.wantCode {
				t.Fatalf("expected error code %s, got %s", tc.wantCode, errResp.Code)
			}
		})
	}
}

func TestAuthHandler_RegisterLocalhostEmailRejectedByBindingHasStructuredLog(t *testing.T) {
	start := time.Now()
	router, _ := newAccountHandlerRouter(false, false, nil)
	body := `{"email":"user@localhost","account":"local_account","name":"Local User","password":"redacted-password","confirm_password":"redacted-password"}`

	t.Log("Given: email uses localhost single-label domain")
	t.Log("Input: email=user@localhost account=local_account name=Local User password_present=true")
	t.Log("Action: POST /api/auth/register")
	t.Log("Note: go-playground/validator email tag rejects single-label domains at the binding layer")

	resp := performAccountRegisterRequest(router, body)
	errResp := decodeAccountHandlerError(t, resp.Body)

	t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
	t.Log("Mutation: register usecase not invoked")
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for localhost email at binding layer, got %d body=%s", resp.Code, resp.Body.String())
	}
	if errResp.Code != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %s", errResp.Code)
	}
}

func TestAuthHandler_LoginSuccessSetsBearerHeaderHasStructuredLog(t *testing.T) {
	start := time.Now()
	router, uow := newAccountHandlerLoginRouter()
	body := `{"identifier":"login@example.com","password":"redacted-password","device_id":"11111111-1111-1111-1111-111111111111"}`

	t.Log("Given: login usecase dependencies succeed")
	t.Log("Input: valid login JSON with identifier/password/device_id")
	t.Log("Action: POST /api/auth/login")

	resp := performAccountLoginRequest(router, body)
	authHeader := resp.Header().Get("Authorization")

	t.Logf("Output: status=%d body=%s bearer_header_present=%t", resp.Code, resp.Body.String(), strings.HasPrefix(authHeader, "Bearer "))
	t.Logf("Mutation: commit_calls=%d", uow.tx.commitCalls)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		t.Fatalf("expected Bearer auth header, got %q", authHeader)
	}
	var out LoginResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
		t.Fatalf("expected valid login response json, got %v", err)
	}
	if out.LoginStatus != "authenticated" {
		t.Fatalf("expected login_status=authenticated, got %+v", out)
	}
}

func TestAuthHandler_LoginInvalidPayloadHasStructuredLog(t *testing.T) {
	start := time.Now()
	router := newAccountHandlerLoginInvalidPayloadRouter()
	body := `{"identifier":"login@example.com","device_id":"11111111-1111-1111-1111-111111111111"}`

	t.Log("Given: invalid login request payload")
	t.Log("Input: missing_password=true")
	t.Log("Action: POST /api/auth/login")

	resp := performAccountLoginRequest(router, body)
	errResp := decodeAccountHandlerError(t, resp.Body)

	t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
	t.Log("Mutation: login usecase not invoked")
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if errResp.Code != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %+v", errResp)
	}
}

func TestAuthHandler_VerifyEmailSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	store.sessions["redacted-token"] = authPort.VerificationSession{
		AccountID:         101,
		Code:              "123456",
		ExpiresAtMS:       time.Now().Add(3 * time.Minute).UnixMilli(),
		RemainingAttempts: 3,
	}
	store.accountTokens[101] = "redacted-token"
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	accountRepo.accountsByID[101] = &domainaccount.Account{
		ID:          101,
		Email:       shared.EmailAddress("new@example.com"),
		AccountName: "new_account",
		Status:      domainaccount.Applying,
	}
	router := newAccountHandlerVerifyEmailRouter(store, accountRepo)

	t.Log("Given: verification token and 6-digit code map to an applying account")
	t.Log("Input: token_present=true code_present=true")
	t.Log("Action: POST /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, `{"token":"redacted-token","code":"123456"}`)
	body := resp.Body.String()
	updatedStatus := domainaccount.Status("<nil>")
	if accountRepo.lastUpdated != nil {
		updatedStatus = accountRepo.lastUpdated.Status
	}

	t.Logf("Output: status=%d body=%s", resp.Code, body)
	t.Logf("Mutation: get_calls=%d delete_calls=%d update_calls=%d token_removed=%t account_status=%s",
		store.getCalls, store.deleteCalls, accountRepo.updateCalls, len(store.sessions) == 0, updatedStatus)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, body)
	}
	if body != "{}" {
		t.Fatalf("expected empty JSON body, got %s", body)
	}
	if store.deleteCalls != 1 || accountRepo.updateCalls != 1 || accountRepo.lastUpdated == nil || accountRepo.lastUpdated.Status != domainaccount.Active {
		t.Fatalf("expected delete=1 update=1 active account, got delete=%d update=%d account=%+v",
			store.deleteCalls, accountRepo.updateCalls, accountRepo.lastUpdated)
	}
}

func TestAuthHandler_VerifyEmailMissingTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	router := newAccountHandlerVerifyEmailRouter(store, accountRepo)

	t.Log("Given: verify email request has an invalid payload")
	t.Log("Input: token_present=false code_present=false")
	t.Log("Action: POST /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, `{}`)
	errResp := decodeAccountHandlerError(t, resp.Body)

	t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
	t.Logf("Mutation: get_calls=%d delete_calls=%d update_calls=%d", store.getCalls, store.deleteCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if errResp.Code != "INVALID_REQUEST" {
		t.Fatalf("expected INVALID_REQUEST, got %+v", errResp)
	}
	if store.getCalls != 0 || store.deleteCalls != 0 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected missing token to stop before usecase, got get=%d delete=%d update=%d",
			store.getCalls, store.deleteCalls, accountRepo.updateCalls)
	}
}

func TestAuthHandler_VerifyEmailInvalidTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	router := newAccountHandlerVerifyEmailRouter(store, accountRepo)

	t.Log("Given: verify email token is not present in the store")
	t.Log("Input: token_present=true code_present=true")
	t.Log("Action: POST /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, `{"token":"missing-token","code":"123456"}`)
	errResp := decodeAccountHandlerError(t, resp.Body)

	t.Logf("Output: status=%d code=%s message=%q", resp.Code, errResp.Code, errResp.Message)
	t.Logf("Mutation: get_calls=%d delete_calls=%d update_calls=%d", store.getCalls, store.deleteCalls, accountRepo.updateCalls)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	if errResp.Code != "TOKEN_INVALID" {
		t.Fatalf("expected TOKEN_INVALID, got %+v", errResp)
	}
	if store.getCalls != 1 || store.deleteCalls != 0 || accountRepo.updateCalls != 0 {
		t.Fatalf("expected invalid token to read store only, got get=%d delete=%d update=%d",
			store.getCalls, store.deleteCalls, accountRepo.updateCalls)
	}
}

var (
	_ transaction.UnitOfWork   = (*accountHandlerUOWStub)(nil)
	_ transaction.Transaction  = (*accountHandlerTxStub)(nil)
	_ domainaccount.Repository = (*accountHandlerAccountRepoStub)(nil)
	_ domainaccount.Repository = (*accountHandlerVerifyEmailAccountRepoStub)(nil)
	_ domainuser.Repository    = (*accountHandlerUserRepoStub)(nil)
	_ userrole.Repository      = (*accountHandlerUserRoleRepoStub)(nil)
	_ domaindevice.Repository  = (*accountHandlerDeviceRepoStub)(nil)
	_ participant.Repository   = (*accountHandlerParticipantRepoStub)(nil)
)
