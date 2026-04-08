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

	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/response"
	"github.com/gin-gonic/gin"
)

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
	err error
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
	return false
}

type accountHandlerAccountRepoStub struct {
	emailExists   bool
	accountExists bool
	createErr     error
}

func (s accountHandlerAccountRepoStub) FindByID(context.Context, shared.AccountID) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (s accountHandlerAccountRepoStub) FindByAccountName(_ context.Context, name string) (*domainaccount.Account, error) {
	if s.accountExists {
		return &domainaccount.Account{ID: 1, AccountName: name, Status: domainaccount.Active}, nil
	}
	return nil, domainaccount.ErrAccountNotFound
}

func (s accountHandlerAccountRepoStub) FindByEmail(_ context.Context, email shared.EmailAddress) (*domainaccount.Account, error) {
	if s.emailExists {
		return &domainaccount.Account{ID: 1, Email: email, Status: domainaccount.Active}, nil
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

type accountHandlerVerificationStoreStub struct{}

func (accountHandlerVerificationStoreStub) Store(context.Context, string, int64, time.Duration) error {
	return nil
}

func (accountHandlerVerificationStoreStub) Get(context.Context, string) (int64, bool, error) {
	return 0, false, nil
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
	tokens      map[string]int64
	getCalls    int
	deleteCalls int
}

func newAccountHandlerVerifyEmailStoreStub() *accountHandlerVerifyEmailStoreStub {
	return &accountHandlerVerifyEmailStoreStub{tokens: map[string]int64{}}
}

func (s *accountHandlerVerifyEmailStoreStub) Store(context.Context, string, int64, time.Duration) error {
	return nil
}

func (s *accountHandlerVerifyEmailStoreStub) Get(_ context.Context, token string) (int64, bool, error) {
	s.getCalls++
	id, ok := s.tokens[token]
	return id, ok, nil
}

func (s *accountHandlerVerifyEmailStoreStub) Delete(_ context.Context, token string) error {
	s.deleteCalls++
	delete(s.tokens, token)
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
	handler := NewAuthHandler(registerUseCase, nil, nil, nil, nil)
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
	handler := NewAuthHandler(nil, nil, nil, nil, verifyEmailUseCase)
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

func performAccountVerifyEmailRequest(router *gin.Engine, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
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

func TestAuthHandler_RegisterSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	router, uow := newAccountHandlerRouter(false, false, nil)
	body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password"}`

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
	if resp.Body.String() != "{}" {
		t.Fatalf("expected empty JSON object, got %s", resp.Body.String())
	}
}

func TestAuthHandler_RegisterInvalidPayloadHasStructuredLog(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "invalid email", body: `{"email":"not-an-email","account":"new_account","name":"New Display","password":"redacted-password"}`},
		{name: "missing fields", body: `{"email":"new@example.com"}`},
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
			body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password"}`

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
	body := `{"email":"new@example.com","account":"new_account","name":"New Display","password":"redacted-password"}`

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

func TestAuthHandler_VerifyEmailSuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	store := newAccountHandlerVerifyEmailStoreStub()
	store.tokens["redacted-token"] = 101
	accountRepo := newAccountHandlerVerifyEmailAccountRepoStub()
	accountRepo.accountsByID[101] = &domainaccount.Account{
		ID:          101,
		Email:       shared.EmailAddress("new@example.com"),
		AccountName: "new_account",
		Status:      domainaccount.Applying,
	}
	router := newAccountHandlerVerifyEmailRouter(store, accountRepo)

	t.Log("Given: verification token maps to an applying account")
	t.Log("Input: token_present=true")
	t.Log("Action: GET /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, "/api/auth/verify-email?token=redacted-token")
	body := resp.Body.String()
	contentType := resp.Header().Get("Content-Type")
	updatedStatus := domainaccount.Status("<nil>")
	if accountRepo.lastUpdated != nil {
		updatedStatus = accountRepo.lastUpdated.Status
	}

	t.Logf("Output: status=%d content_type=%q body_len=%d", resp.Code, contentType, len(body))
	t.Logf("Mutation: get_calls=%d delete_calls=%d update_calls=%d token_removed=%t account_status=%s",
		store.getCalls, store.deleteCalls, accountRepo.updateCalls, len(store.tokens) == 0, updatedStatus)
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, body)
	}
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("expected HTML content type, got %q", contentType)
	}
	for _, want := range []string{
		"tentserv-chat://email-verified",
		"--background: 38 16.7% 90.6%;",
		"--card: 120 3.8% 64.1%;",
		"--primary: 174 12.5% 15.7%;",
		"--radius: 8px;",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected verify email HTML to contain %q", want)
		}
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

	t.Log("Given: verify email request has an empty token")
	t.Log("Input: token_present=false")
	t.Log("Action: GET /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, "/api/auth/verify-email?token=")
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
	t.Log("Input: token_present=true")
	t.Log("Action: GET /api/auth/verify-email")

	resp := performAccountVerifyEmailRequest(router, "/api/auth/verify-email?token=missing-token")
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
)
