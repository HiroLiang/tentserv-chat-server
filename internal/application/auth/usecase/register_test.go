package usecase

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"sync"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/gofrs/uuid"
)

func TestMain(m *testing.M) {
	if err := config.LoadConfig("../../../../dev-doc/config"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type authRegisterTxStub struct {
	commitErr     error
	commitCalls   int
	rollbackCalls int
}

func (s *authRegisterTxStub) Commit() error {
	s.commitCalls++
	return s.commitErr
}

func (s *authRegisterTxStub) Rollback() error {
	s.rollbackCalls++
	return nil
}

type authRegisterUOWStub struct {
	tx         *authRegisterTxStub
	beginErr   error
	beginCalls int
}

func (s *authRegisterUOWStub) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	s.beginCalls++
	if s.beginErr != nil {
		return ctx, nil, s.beginErr
	}
	if s.tx == nil {
		s.tx = &authRegisterTxStub{}
	}
	return ctx, s.tx, nil
}

type authHasherStub struct {
	hash            string
	hashErr         error
	hashCalls       int
	lastPlain       string
	verifyResult    bool
	verifyCalls     int
	lastVerifyPlain string
	lastVerifyHash  string
}

func (s *authHasherStub) Hash(str string) (string, error) {
	s.hashCalls++
	s.lastPlain = str
	if s.hashErr != nil {
		return "", s.hashErr
	}
	if s.hash == "" {
		return "argon2-hash", nil
	}
	return s.hash, nil
}

func (s *authHasherStub) HashBytes(bytes []byte) (string, error) {
	return s.Hash(string(bytes))
}

func (s *authHasherStub) Verify(plain, hash string) bool {
	s.verifyCalls++
	s.lastVerifyPlain = plain
	s.lastVerifyHash = hash
	return s.verifyResult
}

type authAccountRepoStub struct {
	mu              sync.Mutex
	accountsByID    map[shared.AccountID]*account.Account
	accountsByEmail map[shared.EmailAddress]*account.Account
	accountsByName  map[string]*account.Account

	findByIDErr         error
	findByEmailErr      error
	findByAccountErr    error
	createErr           error
	updateErr           error
	registerDeviceErr   error
	recordLoginEventErr error

	nextID                shared.AccountID
	findByIDCalls         int
	findByEmailCalls      int
	findByAccountCalls    int
	createCalls           int
	updateCalls           int
	registerDeviceCalls   int
	recordLoginEventCalls int
	lastFindByEmail       shared.EmailAddress
	lastFindByAccountName string
	lastCreated           *account.Account
	lastUpdated           *account.Account
	lastRegisteredDevice  *account.AccountDevice
	lastLoginEvent        *account.AccountLoginEvent
}

func newAuthAccountRepoStub() *authAccountRepoStub {
	return &authAccountRepoStub{
		accountsByID:    map[shared.AccountID]*account.Account{},
		accountsByEmail: map[shared.EmailAddress]*account.Account{},
		accountsByName:  map[string]*account.Account{},
		nextID:          101,
	}
}

func (s *authAccountRepoStub) FindByID(_ context.Context, id shared.AccountID) (*account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findByIDCalls++
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	acc, ok := s.accountsByID[id]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return cloneAuthAccount(acc), nil
}

func (s *authAccountRepoStub) FindByAccountName(_ context.Context, accountName string) (*account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findByAccountCalls++
	s.lastFindByAccountName = accountName
	if s.findByAccountErr != nil {
		return nil, s.findByAccountErr
	}
	acc, ok := s.accountsByName[accountName]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return cloneAuthAccount(acc), nil
}

func (s *authAccountRepoStub) FindByEmail(_ context.Context, email shared.EmailAddress) (*account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findByEmailCalls++
	s.lastFindByEmail = email
	if s.findByEmailErr != nil {
		return nil, s.findByEmailErr
	}
	acc, ok := s.accountsByEmail[email]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return cloneAuthAccount(acc), nil
}

func (s *authAccountRepoStub) Create(_ context.Context, acc *account.Account) (shared.AccountID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createCalls++
	if s.createErr != nil {
		return 0, s.createErr
	}
	id := s.nextID
	created := cloneAuthAccount(acc)
	created.ID = id
	s.accountsByID[id] = created
	s.accountsByEmail[created.Email] = created
	s.accountsByName[created.AccountName] = created
	s.lastCreated = created
	return id, nil
}

func (s *authAccountRepoStub) Update(_ context.Context, acc *account.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	updated := cloneAuthAccount(acc)
	s.accountsByID[updated.ID] = updated
	s.accountsByEmail[updated.Email] = updated
	s.accountsByName[updated.AccountName] = updated
	s.lastUpdated = updated
	return nil
}

func (s *authAccountRepoStub) RegisterDevice(_ context.Context, device *account.AccountDevice) error {
	s.registerDeviceCalls++
	if s.registerDeviceErr != nil {
		return s.registerDeviceErr
	}
	copied := *device
	s.lastRegisteredDevice = &copied
	return nil
}

func (s *authAccountRepoStub) UpdateDeviceStatus(context.Context, shared.AccountID, shared.DeviceID, account.DeviceStatus) error {
	return nil
}

func (s *authAccountRepoStub) RecordLoginEvent(_ context.Context, event *account.AccountLoginEvent) error {
	s.recordLoginEventCalls++
	if s.recordLoginEventErr != nil {
		return s.recordLoginEventErr
	}
	copied := *event
	s.lastLoginEvent = &copied
	return nil
}

func (s *authAccountRepoStub) ReplaceDevices(context.Context, shared.AccountID, []account.AccountDevice) error {
	return nil
}

type authUserRepoStub struct {
	nextID           shared.UserID
	createErr        error
	updateErr        error
	findByAccountErr error
	createCalls      int
	updateCalls      int
	lastCreated      *user.User
	lastUpdated      *user.User
	usersByAccountID map[shared.AccountID][]user.User
}

func (s *authUserRepoStub) Create(_ context.Context, u *user.User) (shared.UserID, error) {
	s.createCalls++
	if s.createErr != nil {
		return 0, s.createErr
	}
	if s.nextID == 0 {
		s.nextID = 501
	}
	created := *u
	created.ID = s.nextID
	s.lastCreated = &created
	if s.usersByAccountID == nil {
		s.usersByAccountID = map[shared.AccountID][]user.User{}
	}
	s.usersByAccountID[created.AccountID] = append(s.usersByAccountID[created.AccountID], created)
	return s.nextID, nil
}

func (s *authUserRepoStub) FindByID(context.Context, shared.UserID) (*user.User, error) {
	return nil, user.ErrUserNotFound
}

func (s *authUserRepoStub) FindByAccountID(_ context.Context, accountID shared.AccountID) (*[]user.User, error) {
	if s.findByAccountErr != nil {
		return nil, s.findByAccountErr
	}
	users := append([]user.User(nil), s.usersByAccountID[accountID]...)
	return &users, nil
}

func (s *authUserRepoStub) Update(_ context.Context, updatedUser *user.User) error {
	s.updateCalls++
	if s.updateErr != nil {
		return s.updateErr
	}
	copied := *updatedUser
	s.lastUpdated = &copied
	if s.usersByAccountID == nil {
		s.usersByAccountID = map[shared.AccountID][]user.User{}
	}
	users := s.usersByAccountID[updatedUser.AccountID]
	for i := range users {
		if users[i].ID == updatedUser.ID {
			users[i] = copied
			s.usersByAccountID[updatedUser.AccountID] = users
			return nil
		}
	}
	s.usersByAccountID[updatedUser.AccountID] = append(users, copied)
	return nil
}

func (s *authUserRepoStub) SearchByName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (s *authUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

func (s *authUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*user.UserSearchResult, error) {
	return nil, nil
}

type authUserRoleRepoStub struct {
	assignErr   error
	assignCalls int
	assigned    []role.Code
}

func (s *authUserRoleRepoStub) FindRolesByUser(context.Context, shared.UserID) ([]*role.Role, error) {
	return nil, nil
}

func (s *authUserRoleRepoStub) Exists(context.Context, shared.UserID, role.Code) bool {
	return false
}

func (s *authUserRoleRepoStub) Assign(_ context.Context, _ shared.UserID, code role.Code) error {
	s.assignCalls++
	s.assigned = append(s.assigned, code)
	return s.assignErr
}

func (s *authUserRoleRepoStub) Revoke(context.Context, shared.UserID, role.Code) error {
	return nil
}

type authVerificationStoreStub struct {
	mu            sync.Mutex
	sessions      map[string]port.VerificationSession
	accountTokens map[int64]string
	storeErr      error
	getErr        error
	deleteErr     error
	storeCalls    int
	getCalls      int
	deleteCalls   int
	storedToken   string
	storedID      int64
	storedTTL     time.Duration
	storedSession port.VerificationSession
	deleted       []string
}

func newAuthVerificationStoreStub() *authVerificationStoreStub {
	return &authVerificationStoreStub{
		sessions:      map[string]port.VerificationSession{},
		accountTokens: map[int64]string{},
	}
}

func (s *authVerificationStoreStub) Store(_ context.Context, token string, session port.VerificationSession, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storeCalls++
	if s.storeErr != nil {
		return s.storeErr
	}
	s.storedToken = token
	s.storedID = session.AccountID
	s.storedTTL = ttl
	s.storedSession = session
	s.sessions[token] = session
	s.accountTokens[session.AccountID] = token
	return nil
}

func (s *authVerificationStoreStub) Get(_ context.Context, token string) (port.VerificationSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	if s.getErr != nil {
		return port.VerificationSession{}, false, s.getErr
	}
	session, ok := s.sessions[token]
	return session, ok, nil
}

func (s *authVerificationStoreStub) FindTokenByAccountID(_ context.Context, accountID int64) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.accountTokens[accountID]
	if !ok {
		return "", false, nil
	}

	session, sessionOK := s.sessions[token]
	if !sessionOK || session.ExpiresAtMS <= time.Now().UTC().UnixMilli() {
		delete(s.accountTokens, accountID)
		delete(s.sessions, token)
		return "", false, nil
	}

	return token, true, nil
}

func (s *authVerificationStoreStub) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCalls++
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, token)
	session, ok := s.sessions[token]
	if ok {
		delete(s.accountTokens, session.AccountID)
	}
	delete(s.sessions, token)
	return nil
}

type authEmailServiceStub struct {
	sendErr     error
	sendCalls   int
	lastBuilder appEmail.EmailBuilder
}

func (s *authEmailServiceStub) Send(_ context.Context, builder appEmail.EmailBuilder) error {
	s.sendCalls++
	s.lastBuilder = builder
	return s.sendErr
}

type authEmailBuilderStub struct{}

func (authEmailBuilderStub) BuildEmail(context.Context) (*shared.Email, error) {
	return &shared.Email{}, nil
}

type registerMailFactoryCapture struct {
	calls            int
	recipientEmail   string
	recipientName    string
	verificationCode string
}

func (c *registerMailFactoryCapture) factory(recipientEmail, recipientName, verificationCode string) appEmail.EmailBuilder {
	c.calls++
	c.recipientEmail = recipientEmail
	c.recipientName = recipientName
	c.verificationCode = verificationCode
	return authEmailBuilderStub{}
}

func newAuthRegisterUseCase(
	uow *authRegisterUOWStub,
	hasher *authHasherStub,
	accountRepo *authAccountRepoStub,
	userRepo *authUserRepoStub,
	roleRepo *authUserRoleRepoStub,
	store *authVerificationStoreStub,
	emailService *authEmailServiceStub,
	factory *registerMailFactoryCapture,
) *RegisterUseCase {
	return NewRegisterUseCase(uow, hasher, accountRepo, userRepo, roleRepo, store, emailService, factory.factory)
}

func authRegisterInput(email, accountName, displayName, password string) appShared.UseCaseInput[RegisterInput] {
	return appShared.UseCaseInput[RegisterInput]{
		Data: RegisterInput{
			Email:           email,
			Account:         accountName,
			Name:            displayName,
			Password:        password,
			ConfirmPassword: password,
		},
	}
}

func newExistingAuthAccount(id shared.AccountID, email shared.EmailAddress, accountName string, status account.Status) *account.Account {
	return &account.Account{
		ID:          id,
		PublicID:    uuid.Nil,
		Email:       email,
		AccountName: accountName,
		Password:    "existing-hash",
		Status:      status,
		UserLimit:   1,
	}
}

func cloneAuthAccount(acc *account.Account) *account.Account {
	if acc == nil {
		return nil
	}
	cloned := *acc
	cloned.UserIDs = append([]shared.UserID(nil), acc.UserIDs...)
	cloned.Devices = append([]account.AccountDevice(nil), acc.Devices...)
	return &cloned
}

func assertRegisterError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, got %v", want, got)
	}
}

func TestRegisterUseCase_SuccessHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{hash: "hashed-password"}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{nextID: 501}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	mailFactory := &registerMailFactoryCapture{}
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, userRepo, roleRepo, store, emailService, mailFactory)
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: no existing account rows and all dependencies succeed")
	t.Logf("Input: email=%s account=%s display_name=%q password_present=%t", input.Data.Email, input.Data.Account, input.Data.Name, input.Data.Password != "")
	t.Log("Action: execute account registration happy path")

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Output: unexpected error=%v; Duration=%s", err, time.Since(start))
	}

	t.Logf("Output: account_id=%d verification_token_present=%t verification_code_present=%t", out.ID, store.storedToken != "", mailFactory.verificationCode != "")
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d user_create_calls=%d role_assign_calls=%d account_update_calls=%d store_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, userRepo.createCalls,
		roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	if out.ID != 101 {
		t.Fatalf("expected account id 101, got %d", out.ID)
	}
	if accountRepo.lastCreated == nil || accountRepo.lastCreated.Password != "hashed-password" || accountRepo.lastCreated.Status != account.Applying {
		t.Fatalf("account was not created with hash/applying status: %+v", accountRepo.lastCreated)
	}
	if userRepo.lastCreated == nil || userRepo.lastCreated.Name != "New Display" || userRepo.lastCreated.AccountID != 101 {
		t.Fatalf("user was not created from display name: %+v", userRepo.lastCreated)
	}
	if len(roleRepo.assigned) != 1 || roleRepo.assigned[0] != role.User {
		t.Fatalf("expected default user role assignment, got %v", roleRepo.assigned)
	}
	if accountRepo.lastUpdated == nil || len(accountRepo.lastUpdated.UserIDs) != 1 || accountRepo.lastUpdated.UserIDs[0] != 501 {
		t.Fatalf("expected account to link user 501, got %+v", accountRepo.lastUpdated)
	}
	if store.storedID != 101 || store.storedTTL != config.App().Email.VerifyTTL {
		t.Fatalf("expected verification token stored for account 101 with config ttl, got id=%d ttl=%s", store.storedID, store.storedTTL)
	}
	if mailFactory.recipientEmail != input.Data.Email || mailFactory.recipientName != input.Data.Name {
		t.Fatalf("expected register mail factory to use email/display name, got email=%s name=%s", mailFactory.recipientEmail, mailFactory.recipientName)
	}
	if len(mailFactory.verificationCode) != 6 {
		t.Fatalf("expected verification code to be 6 digits, got %q", mailFactory.verificationCode)
	}
	if uow.tx.commitCalls != 1 || uow.tx.rollbackCalls != 0 {
		t.Fatalf("expected commit=1 rollback=0, got commit=%d rollback=%d", uow.tx.commitCalls, uow.tx.rollbackCalls)
	}
}

func TestRegisterUseCase_InvalidEmailHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("not-an-email", "new_account", "New Display", "redacted-password")

	t.Log("Given: malformed email input")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration email validation path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrInvalidEmail)
	if accountRepo.findByEmailCalls != 0 || hasher.hashCalls != 0 || accountRepo.createCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected no precheck/hash/create/commit, got find_email=%d hash=%d create=%d commit=%d",
			accountRepo.findByEmailCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_EmailExistsHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	email := shared.EmailAddress("taken@example.com")
	accountRepo.accountsByEmail[email] = newExistingAuthAccount(1, email, "taken_account", account.Active)
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput(string(email), "new_account", "New Display", "redacted-password")

	t.Log("Given: account repository already contains the requested email")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration email conflict path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrEmailExist)
	if accountRepo.findByAccountCalls != 0 || hasher.hashCalls != 0 || accountRepo.createCalls != 0 {
		t.Fatalf("expected email precheck to stop flow, got find_account=%d hash=%d create=%d", accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls)
	}
}

func TestRegisterUseCase_AccountExistsHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	email := shared.EmailAddress("taken@example.com")
	accountRepo.accountsByName["taken_account"] = newExistingAuthAccount(1, email, "taken_account", account.Active)
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "taken_account", "New Display", "redacted-password")

	t.Log("Given: account repository already contains the requested account name")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration account conflict path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrAccountExist)
	if hasher.hashCalls != 0 || accountRepo.createCalls != 0 {
		t.Fatalf("expected account precheck to stop flow, got hash=%d create=%d", hasher.hashCalls, accountRepo.createCalls)
	}
}

func TestRegisterUseCase_HasherFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{hashErr: errors.New("argon2 unavailable")}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: hasher returns an error after account prechecks pass")
	t.Logf("Input: email=%s account=%s display_name=%q password_present=%t", input.Data.Email, input.Data.Account, input.Data.Name, input.Data.Password != "")
	t.Log("Action: execute account registration password hash failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrInvalidPassword)
	if accountRepo.createCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected no create/commit on hash failure, got create=%d commit=%d", accountRepo.createCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_AccountCreateUniqueRaceHasStructuredLog(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "email unique race", err: account.ErrEmailExist, wantErr: ErrEmailExist},
		{name: "account unique race", err: account.ErrAccountExist, wantErr: ErrAccountExist},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			uow := &authRegisterUOWStub{}
			hasher := &authHasherStub{}
			accountRepo := newAuthAccountRepoStub()
			accountRepo.createErr = tc.err
			uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
			input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

			t.Logf("Given: prechecks pass but account create returns %v", tc.err)
			t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
			t.Log("Action: execute account registration DB unique race path")

			out, err := uc.Execute(context.Background(), input)

			t.Logf("Output: out=%+v err=%v", out, err)
			t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d user_create_calls=%d commit_calls=%d rollback_calls=%d",
				accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, 0, uow.tx.commitCalls, uow.tx.rollbackCalls)
			t.Logf("Duration: %s", time.Since(start))

			assertRegisterError(t, err, tc.wantErr)
			if accountRepo.createCalls != 1 || uow.tx.commitCalls != 0 {
				t.Fatalf("expected create=1 commit=0, got create=%d commit=%d", accountRepo.createCalls, uow.tx.commitCalls)
			}
		})
	}
}

func TestRegisterUseCase_UserCreateFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{createErr: errors.New("insert user failed")}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, &authUserRoleRepoStub{}, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: account create succeeds but user create returns an error")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration user create failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d store_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.createCalls, userRepo.createCalls, 0, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if accountRepo.createCalls != 1 || userRepo.createCalls != 1 || store.storeCalls != 0 || emailService.sendCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected DB flow to stop before external effects, got account_create=%d user_create=%d store=%d email=%d commit=%d",
			accountRepo.createCalls, userRepo.createCalls, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_RoleAssignFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{}
	roleRepo := &authUserRoleRepoStub{assignErr: errors.New("role insert failed")}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, roleRepo, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: user create succeeds but default role assignment returns an error")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration role assign failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d account_update_calls=%d store_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.createCalls, userRepo.createCalls, roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if roleRepo.assignCalls != 1 || accountRepo.updateCalls != 0 || store.storeCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected role failure before account update/external effects, got role=%d update=%d store=%d commit=%d",
			roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_AccountUpdateFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	accountRepo := newAuthAccountRepoStub()
	accountRepo.updateErr = errors.New("update account failed")
	userRepo := &authUserRepoStub{}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, roleRepo, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: account/user/role creation succeeds but account link update returns an error")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration account update failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d account_update_calls=%d store_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.createCalls, userRepo.createCalls, roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if accountRepo.updateCalls != 1 || store.storeCalls != 0 || emailService.sendCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected update failure before external effects, got update=%d store=%d email=%d commit=%d",
			accountRepo.updateCalls, store.storeCalls, emailService.sendCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_TokenStoreFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	store.storeErr = errors.New("redis unavailable")
	emailService := &authEmailServiceStub{}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, roleRepo, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: DB transaction commits but verification token store returns an error")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration token store failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d account_update_calls=%d store_calls=%d delete_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.createCalls, userRepo.createCalls, roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, store.deleteCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if uow.tx.commitCalls != 1 || store.storeCalls != 1 || store.deleteCalls != 0 || emailService.sendCalls != 0 || uow.tx.rollbackCalls != 0 {
		t.Fatalf("expected committed DB and no email/delete on store failure, got commit=%d store=%d delete=%d email=%d rollback=%d",
			uow.tx.commitCalls, store.storeCalls, store.deleteCalls, emailService.sendCalls, uow.tx.rollbackCalls)
	}
}

func TestRegisterUseCase_EmailSendFailureCleansTokenHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{sendErr: errors.New("resend unavailable")}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, roleRepo, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: verification token stores successfully but email sending fails")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration email send failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: store_calls=%d delete_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d token_cleaned=%t",
		store.storeCalls, store.deleteCalls, emailService.sendCalls, uow.tx.commitCalls, uow.tx.rollbackCalls, len(store.sessions) == 0)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if uow.tx.commitCalls != 1 || store.storeCalls != 1 || store.deleteCalls != 1 || emailService.sendCalls != 1 || len(store.sessions) != 0 {
		t.Fatalf("expected committed DB and token cleanup on email failure, got commit=%d store=%d delete=%d email=%d remaining_tokens=%d",
			uow.tx.commitCalls, store.storeCalls, store.deleteCalls, emailService.sendCalls, len(store.sessions))
	}
}

func TestRegisterUseCase_CommitFailureHasStructuredLog(t *testing.T) {
	start := time.Now()
	tx := &authRegisterTxStub{commitErr: errors.New("commit failed")}
	uow := &authRegisterUOWStub{tx: tx}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	uc := newAuthRegisterUseCase(uow, &authHasherStub{}, accountRepo, userRepo, roleRepo, store, emailService, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "redacted-password")

	t.Log("Given: DB mutation path succeeds but transaction commit returns an error")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration commit failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: account_create_calls=%d user_create_calls=%d role_assign_calls=%d account_update_calls=%d store_calls=%d email_send_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.createCalls, userRepo.createCalls, roleRepo.assignCalls, accountRepo.updateCalls, store.storeCalls, emailService.sendCalls, tx.commitCalls, tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrRegisterFailed)
	if tx.commitCalls != 1 || tx.rollbackCalls != 1 || store.storeCalls != 0 || emailService.sendCalls != 0 {
		t.Fatalf("expected rollback and no external effects on commit failure, got commit=%d rollback=%d store=%d email=%d",
			tx.commitCalls, tx.rollbackCalls, store.storeCalls, emailService.sendCalls)
	}
}

func TestRegisterUseCase_LongEmailHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	longEmail := strings.Repeat("a", 243) + "@example.com" // 255 chars total
	input := authRegisterInput(longEmail, "new_account", "New Display", "redacted-password")

	t.Log("Given: email string exceeds RFC 5321 maximum length of 254 characters")
	t.Logf("Input: email_len=%d account=%s display_name=%q", len(input.Data.Email), input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration with oversized email")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrInvalidEmail)
	if accountRepo.findByEmailCalls != 0 || hasher.hashCalls != 0 || accountRepo.createCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected long email to stop flow before repo calls, got find_email=%d hash=%d create=%d commit=%d",
			accountRepo.findByEmailCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_InvalidAccountNameHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "invalid account!", "New Display", "redacted-password")

	t.Log("Given: account name contains disallowed characters (spaces and punctuation)")
	t.Logf("Input: email=%s account=%q display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration with invalid account name format")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrInvalidAccount)
	if accountRepo.findByAccountCalls != 0 || hasher.hashCalls != 0 || accountRepo.createCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected invalid account name to stop flow, got find_account=%d hash=%d create=%d commit=%d",
			accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_CommonPasswordHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "password")

	t.Log("Given: password is in the common/weak password list")
	t.Logf("Input: email=%s account=%s display_name=%q password_present=true", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration with common password")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrWeakPassword)
	if hasher.hashCalls != 0 || accountRepo.createCalls != 0 || uow.tx.commitCalls != 0 {
		t.Fatalf("expected common password to stop flow before hashing, got hash=%d create=%d commit=%d",
			hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls)
	}
}

func TestRegisterUseCase_CommonPasswordCaseInsensitiveHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{}
	accountRepo := newAuthAccountRepoStub()
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, &authUserRepoStub{}, &authUserRoleRepoStub{}, newAuthVerificationStoreStub(), &authEmailServiceStub{}, &registerMailFactoryCapture{})
	input := authRegisterInput("new@example.com", "new_account", "New Display", "PASSWORD")

	t.Log("Given: password matches a common password when lowercased")
	t.Logf("Input: email=%s account=%s display_name=%q password_present=true", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration with uppercased common password")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: out=%+v err=%v", out, err)
	t.Logf("Mutation: hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertRegisterError(t, err, ErrWeakPassword)
	if hasher.hashCalls != 0 {
		t.Fatalf("expected common password check to run before hashing, got hash_calls=%d", hasher.hashCalls)
	}
}

func TestRegisterUseCase_LocalhostEmailAcceptedHasStructuredLog(t *testing.T) {
	start := time.Now()
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{hash: "hashed-password"}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{nextID: 501}
	roleRepo := &authUserRoleRepoStub{}
	store := newAuthVerificationStoreStub()
	emailService := &authEmailServiceStub{}
	mailFactory := &registerMailFactoryCapture{}
	uc := newAuthRegisterUseCase(uow, hasher, accountRepo, userRepo, roleRepo, store, emailService, mailFactory)
	input := authRegisterInput("user@localhost", "local_account", "Local User", "redacted-password")

	t.Log("Given: email uses a single-label domain (localhost) which is RFC 5322 valid")
	t.Log("Note: the HTTP binding layer (go-playground/validator) rejects user@localhost with 400 INVALID_REQUEST.")
	t.Log("      This test validates the usecase behavior when the binding layer is bypassed (e.g. internal calls).")
	t.Logf("Input: email=%s account=%s display_name=%q", input.Data.Email, input.Data.Account, input.Data.Name)
	t.Log("Action: execute account registration with user@localhost email directly at usecase layer")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: account_id=%d err=%v", out.ID, err)
	t.Logf("Mutation: find_email_calls=%d hash_calls=%d account_create_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.findByEmailCalls, hasher.hashCalls, accountRepo.createCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected user@localhost to be accepted as valid email, got err=%v", err)
	}
	if out.ID == 0 {
		t.Fatalf("expected non-zero account ID, got 0")
	}
}

var (
	_ transaction.UnitOfWork  = (*authRegisterUOWStub)(nil)
	_ transaction.Transaction = (*authRegisterTxStub)(nil)
	_ account.Repository      = (*authAccountRepoStub)(nil)
	_ user.Repository         = (*authUserRepoStub)(nil)
	_ userrole.Repository     = (*authUserRoleRepoStub)(nil)
	_ port.VerificationStore  = (*authVerificationStoreStub)(nil)
	_ appEmail.EmailService   = (*authEmailServiceStub)(nil)
	_ appEmail.EmailBuilder   = (*authEmailBuilderStub)(nil)
)
