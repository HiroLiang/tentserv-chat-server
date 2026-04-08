package usecase

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

const loginDeviceID = "11111111-1111-1111-1111-111111111111"

type authSessionManagerStub struct {
	createErr   error
	createCalls int
	lastInput   auth.CreateSessionInput
	tokenPair   auth.TokenPair
}

func (s *authSessionManagerStub) Create(_ context.Context, input auth.CreateSessionInput) (auth.TokenPair, error) {
	s.createCalls++
	s.lastInput = input
	if s.createErr != nil {
		return auth.TokenPair{}, s.createErr
	}
	if s.tokenPair.AccessToken == "" {
		s.tokenPair = auth.TokenPair{
			AccessToken:  auth.AccessToken("access-token"),
			RefreshToken: auth.RefreshToken("refresh-token"),
			ExpiresAt:    time.Now().Add(time.Hour),
		}
	}
	return s.tokenPair, nil
}

func (s *authSessionManagerStub) FindByToken(context.Context, auth.AccessToken) (*auth.Session, error) {
	return nil, auth.ErrSessionNotFound
}

func (s *authSessionManagerStub) Refresh(context.Context, auth.RefreshToken) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *authSessionManagerStub) Revoke(context.Context, auth.AccessToken) error { return nil }

func (s *authSessionManagerStub) RevokeAllForUser(context.Context, shared.AccountID) error {
	return nil
}

func (s *authSessionManagerStub) RevokeAll(context.Context) error { return nil }

func (s *authSessionManagerStub) SwitchUser(context.Context, auth.AccessToken, shared.UserID) error {
	return nil
}

type authDeviceRepoStub struct {
	devices       map[string]*device.Device
	findByIDErr   error
	findByIDCalls int
	lastFindID    shared.DeviceID
}

func newAuthDeviceRepoStub() *authDeviceRepoStub {
	return &authDeviceRepoStub{devices: map[string]*device.Device{}}
}

func (s *authDeviceRepoStub) seed(id shared.DeviceID, name string) {
	s.devices[id.String()] = &device.Device{
		ID:       id,
		Platform: device.MacOS,
		Name:     name,
	}
}

func (s *authDeviceRepoStub) FindByID(_ context.Context, id shared.DeviceID) (*device.Device, error) {
	s.findByIDCalls++
	s.lastFindID = id
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	d, ok := s.devices[id.String()]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	copied := *d
	return &copied, nil
}

func (s *authDeviceRepoStub) FindAllByAccountID(context.Context, shared.AccountID) ([]*device.Device, error) {
	return nil, nil
}

func (s *authDeviceRepoStub) Create(context.Context, *device.Device) error { return nil }

func (s *authDeviceRepoStub) Update(context.Context, *device.Device) error { return nil }

func (s *authDeviceRepoStub) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (s *authDeviceRepoStub) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

type authParticipantRepoStub struct {
	participantsByUser map[shared.UserID]*participant.Participant
	findErr            error
	createErr          error
	nextID             participant.ID
	findByUserCalls    int
	createCalls        int
	lastCreated        *participant.Participant
}

func newAuthParticipantRepoStub() *authParticipantRepoStub {
	return &authParticipantRepoStub{
		participantsByUser: map[shared.UserID]*participant.Participant{},
		nextID:             700,
	}
}

func (s *authParticipantRepoStub) seedUser(userID shared.UserID) {
	s.nextID++
	uid := userID
	s.participantsByUser[userID] = &participant.Participant{
		ID:     s.nextID,
		Type:   participant.UserType,
		UserID: &uid,
	}
}

func (s *authParticipantRepoStub) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *authParticipantRepoStub) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	s.findByUserCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	p, ok := s.participantsByUser[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (s *authParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *authParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *authParticipantRepoStub) Create(_ context.Context, p *participant.Participant) error {
	s.createCalls++
	if s.createErr != nil {
		return s.createErr
	}
	s.nextID++
	created := *p
	created.ID = s.nextID
	created.CreatedAt = time.Now()
	s.lastCreated = &created
	if created.UserID != nil {
		s.participantsByUser[*created.UserID] = &created
	}
	p.ID = created.ID
	p.CreatedAt = created.CreatedAt
	return nil
}

type loginMailFactoryCapture struct {
	calls          int
	recipientEmail string
	recipientName  string
	deviceName     string
	deviceID       string
	ip             string
}

func (c *loginMailFactoryCapture) factory(recipientEmail, recipientName, deviceName, deviceID, ip string, _ time.Time) appEmail.EmailBuilder {
	c.calls++
	c.recipientEmail = recipientEmail
	c.recipientName = recipientName
	c.deviceName = deviceName
	c.deviceID = deviceID
	c.ip = ip
	return authEmailBuilderStub{}
}

type loginEmailServiceStub struct {
	sent chan appEmail.EmailBuilder
}

func (s loginEmailServiceStub) Send(_ context.Context, builder appEmail.EmailBuilder) error {
	s.sent <- builder
	return nil
}

func parseLoginDeviceID(t *testing.T) shared.DeviceID {
	t.Helper()
	deviceID, err := shared.ParseDeviceID(loginDeviceID)
	if err != nil {
		t.Fatal(err)
	}
	return deviceID
}

func seedLoginAccount(repo *authAccountRepoStub, identifier string, status account.Status, userIDs ...shared.UserID) *account.Account {
	email := shared.EmailAddress("login@example.com")
	acc := newExistingAuthAccount(100, email, "login_account", status)
	acc.Password = "stored-hash"
	acc.UserIDs = append([]shared.UserID(nil), userIDs...)
	repo.accountsByID[acc.ID] = acc
	repo.accountsByEmail[email] = acc
	repo.accountsByName[acc.AccountName] = acc
	if identifier != "" && identifier != string(email) && identifier != acc.AccountName {
		repo.accountsByName[identifier] = acc
	}
	return acc
}

func newLoginUseCaseFixture(t *testing.T) (*LoginUseCase, *authRegisterUOWStub, *authHasherStub, *authAccountRepoStub, *authUserRepoStub, *authUserRoleRepoStub, *authDeviceRepoStub, *authParticipantRepoStub, *authSessionManagerStub, *loginMailFactoryCapture) {
	t.Helper()
	t.Setenv("APP_ENV", "dev")

	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{verifyResult: true}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{nextID: 501}
	roleRepo := &authUserRoleRepoStub{}
	deviceRepo := newAuthDeviceRepoStub()
	participantRepo := newAuthParticipantRepoStub()
	sessionManager := &authSessionManagerStub{}
	emailService := &authEmailServiceStub{}
	factory := &loginMailFactoryCapture{}
	deviceRepo.seed(parseLoginDeviceID(t), "Hiro's Mac")

	uc := NewLoginUseCase(
		uow,
		hasher,
		sessionManager,
		accountRepo,
		userRepo,
		roleRepo,
		deviceRepo,
		participantRepo,
		emailService,
		factory.factory,
	)
	return uc, uow, hasher, accountRepo, userRepo, roleRepo, deviceRepo, participantRepo, sessionManager, factory
}

func loginInput(identifier, password, deviceID string) *appShared.UseCaseInput[LoginInput] {
	return &appShared.UseCaseInput[LoginInput]{
		Base: appShared.BaseContext{
			Request: appShared.RequestContext{
				IP:        net.ParseIP("203.0.113.10"),
				UserAgent: "TentservDesktop/1.0",
			},
		},
		Data: LoginInput{
			Identifier: identifier,
			Password:   password,
			DeviceID:   deviceID,
		},
	}
}

func assertLoginError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, got %v", want, got)
	}
}

func TestLoginUseCase_EmailLoginCreatesSessionParticipantAndAuditLogHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, uow, hasher, accountRepo, userRepo, roleRepo, deviceRepo, participantRepo, sessionManager, mailFactory := newLoginUseCaseFixture(t)
	seedLoginAccount(accountRepo, "login@example.com", account.Active)
	input := loginInput("login@example.com", "redacted-password", loginDeviceID)

	t.Log("Given: active account has no user participant and startup device exists")
	t.Logf("Input: identifier_type=email password_present=%t device_id=%s user_agent_present=%t", input.Data.Password != "", input.Data.DeviceID, input.Base.Request.UserAgent != "")
	t.Log("Action: execute login usecase happy path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d verify_calls=%d user_create_calls=%d role_assign_calls=%d device_find_calls=%d participant_find_calls=%d participant_create_calls=%d register_device_calls=%d session_create_calls=%d login_event_calls=%d account_update_calls=%d commit_calls=%d rollback_calls=%d email_factory_calls=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, hasher.verifyCalls, userRepo.createCalls, roleRepo.assignCalls, deviceRepo.findByIDCalls,
		participantRepo.findByUserCalls, participantRepo.createCalls, accountRepo.registerDeviceCalls, sessionManager.createCalls,
		accountRepo.recordLoginEventCalls, accountRepo.updateCalls, uow.tx.commitCalls, uow.tx.rollbackCalls, mailFactory.calls)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.TokenPair.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if hasher.lastVerifyPlain != input.Data.Password || hasher.lastVerifyHash != "stored-hash" {
		t.Fatalf("expected password verification against stored hash, got plain=%q hash=%q", hasher.lastVerifyPlain, hasher.lastVerifyHash)
	}
	if userRepo.createCalls != 1 || roleRepo.assignCalls != 1 {
		t.Fatalf("expected default user and role creation, got user=%d roles=%d", userRepo.createCalls, roleRepo.assignCalls)
	}
	if participantRepo.createCalls != 1 || participantRepo.lastCreated == nil || *participantRepo.lastCreated.UserID != 501 {
		t.Fatalf("expected participant for user 501, got calls=%d participant=%+v", participantRepo.createCalls, participantRepo.lastCreated)
	}
	if accountRepo.lastRegisteredDevice == nil || accountRepo.lastRegisteredDevice.DeviceID.String() != loginDeviceID {
		t.Fatalf("expected registered login device, got %+v", accountRepo.lastRegisteredDevice)
	}
	if sessionManager.lastInput.AccountID != 100 || sessionManager.lastInput.UserID != 501 || sessionManager.lastInput.DeviceID.String() != loginDeviceID {
		t.Fatalf("expected session for account 100 user 501 device %s, got %+v", loginDeviceID, sessionManager.lastInput)
	}
	if accountRepo.lastLoginEvent == nil || !accountRepo.lastLoginEvent.Success || accountRepo.lastLoginEvent.UserAgent != "TentservDesktop/1.0" {
		t.Fatalf("expected successful login event with user agent, got %+v", accountRepo.lastLoginEvent)
	}
	if accountRepo.lastUpdated == nil || len(accountRepo.lastUpdated.UserIDs) != 1 || accountRepo.lastUpdated.UserIDs[0] != 501 {
		t.Fatalf("expected account update to link user 501, got %+v", accountRepo.lastUpdated)
	}
	if uow.tx.commitCalls != 1 || uow.tx.rollbackCalls != 0 || mailFactory.calls != 0 {
		t.Fatalf("expected commit=1 rollback=0 no dev email, got commit=%d rollback=%d email=%d", uow.tx.commitCalls, uow.tx.rollbackCalls, mailFactory.calls)
	}
}

func TestLoginUseCase_AccountIdentifierReusesExistingUserAndParticipantHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, _, _, accountRepo, userRepo, roleRepo, _, participantRepo, sessionManager, _ := newLoginUseCaseFixture(t)
	seedLoginAccount(accountRepo, "login_account", account.Active, 777)
	participantRepo.seedUser(777)
	input := loginInput("login_account", "redacted-password", loginDeviceID)

	t.Log("Given: active account has existing current user and participant")
	t.Log("Input: identifier_type=account password_present=true")
	t.Log("Action: execute login usecase account-name path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: find_email_calls=%d find_account_calls=%d user_create_calls=%d role_assign_calls=%d participant_create_calls=%d session_user_id=%d",
		accountRepo.findByEmailCalls, accountRepo.findByAccountCalls, userRepo.createCalls, roleRepo.assignCalls, participantRepo.createCalls, sessionManager.lastInput.UserID)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if accountRepo.findByEmailCalls != 0 || accountRepo.findByAccountCalls != 1 {
		t.Fatalf("expected account-name lookup only, got email=%d account=%d", accountRepo.findByEmailCalls, accountRepo.findByAccountCalls)
	}
	if userRepo.createCalls != 0 || roleRepo.assignCalls != 0 || participantRepo.createCalls != 0 {
		t.Fatalf("expected no duplicate user/role/participant create, got user=%d role=%d participant=%d", userRepo.createCalls, roleRepo.assignCalls, participantRepo.createCalls)
	}
	if sessionManager.lastInput.UserID != 777 {
		t.Fatalf("expected existing user 777 as current user, got %d", sessionManager.lastInput.UserID)
	}
}

func TestLoginUseCase_ParticipantDuplicateRaceIsAcceptedHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, _, _, accountRepo, _, _, _, participantRepo, _, _ := newLoginUseCaseFixture(t)
	seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
	participantRepo.createErr = participant.ErrAlreadyExists
	input := loginInput("login@example.com", "redacted-password", loginDeviceID)

	t.Log("Given: participant lookup misses but create reports an already-existing row")
	t.Log("Input: identifier_type=email password_present=true")
	t.Log("Action: execute login usecase participant race path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: participant_find_calls=%d participant_create_calls=%d login_event_calls=%d", participantRepo.findByUserCalls, participantRepo.createCalls, accountRepo.recordLoginEventCalls)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected duplicate participant race to succeed, got %v", err)
	}
	if participantRepo.createCalls != 1 || accountRepo.recordLoginEventCalls != 1 {
		t.Fatalf("expected participant create race and successful login event, got participant=%d event=%d", participantRepo.createCalls, accountRepo.recordLoginEventCalls)
	}
}

func TestLoginUseCase_RejectsPasswordErrorBeforeSideEffectsHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, uow, hasher, accountRepo, _, _, _, participantRepo, sessionManager, _ := newLoginUseCaseFixture(t)
	hasher.verifyResult = false
	seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
	input := loginInput("login@example.com", "wrong-password", loginDeviceID)

	t.Log("Given: active account exists but password verification fails")
	t.Log("Input: identifier_type=email password_present=true")
	t.Log("Action: execute login usecase password failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: verify_calls=%d participant_create_calls=%d register_device_calls=%d session_create_calls=%d login_event_calls=%d commit_calls=%d rollback_calls=%d",
		hasher.verifyCalls, participantRepo.createCalls, accountRepo.registerDeviceCalls, sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertLoginError(t, err, ErrPasswordError)
	if sessionManager.createCalls != 0 || accountRepo.recordLoginEventCalls != 0 || uow.tx.commitCalls != 0 || uow.tx.rollbackCalls != 1 {
		t.Fatalf("expected rollback before side effects, got session=%d event=%d commit=%d rollback=%d",
			sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	}
}

func TestLoginUseCase_MapsAccountStatusesHasStructuredLog(t *testing.T) {
	cases := []struct {
		name    string
		status  account.Status
		wantErr error
	}{
		{name: "applying", status: account.Applying, wantErr: ErrAccountApplying},
		{name: "inactive", status: account.Inactive, wantErr: ErrAccountInactive},
		{name: "banned", status: account.Banned, wantErr: ErrAccountBanned},
		{name: "deleted", status: account.Deleted, wantErr: ErrAccountNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			uc, uow, _, accountRepo, _, _, _, _, sessionManager, _ := newLoginUseCaseFixture(t)
			seedLoginAccount(accountRepo, "login@example.com", tc.status, 777)
			input := loginInput("login@example.com", "redacted-password", loginDeviceID)

			t.Logf("Given: account status is %s", tc.status)
			t.Log("Input: identifier_type=email")
			t.Log("Action: execute login usecase account-status path")

			out, err := uc.Execute(context.Background(), input)

			t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
			t.Logf("Mutation: session_create_calls=%d login_event_calls=%d commit_calls=%d rollback_calls=%d", sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
			t.Logf("Duration: %s", time.Since(start))

			assertLoginError(t, err, tc.wantErr)
			if sessionManager.createCalls != 0 || accountRepo.recordLoginEventCalls != 0 || uow.tx.commitCalls != 0 || uow.tx.rollbackCalls != 1 {
				t.Fatalf("expected status check to stop flow, got session=%d event=%d commit=%d rollback=%d",
					sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
			}
		})
	}
}

func TestLoginUseCase_RejectsInvalidOrUnknownDeviceHasStructuredLog(t *testing.T) {
	cases := []struct {
		name     string
		deviceID string
		clearMap bool
	}{
		{name: "invalid uuid", deviceID: "not-a-device-id"},
		{name: "unknown uuid", deviceID: loginDeviceID, clearMap: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			uc, uow, _, accountRepo, _, _, deviceRepo, _, sessionManager, _ := newLoginUseCaseFixture(t)
			seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
			if tc.clearMap {
				deviceRepo.devices = map[string]*device.Device{}
			}
			input := loginInput("login@example.com", "redacted-password", tc.deviceID)

			t.Log("Given: login device is invalid or not registered")
			t.Logf("Input: case=%s device_id=%s", tc.name, tc.deviceID)
			t.Log("Action: execute login usecase device validation path")

			out, err := uc.Execute(context.Background(), input)

			t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
			t.Logf("Mutation: device_find_calls=%d session_create_calls=%d login_event_calls=%d commit_calls=%d rollback_calls=%d",
				deviceRepo.findByIDCalls, sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
			t.Logf("Duration: %s", time.Since(start))

			assertLoginError(t, err, ErrInvalidDeviceID)
			if sessionManager.createCalls != 0 || accountRepo.recordLoginEventCalls != 0 || uow.tx.commitCalls != 0 || uow.tx.rollbackCalls != 1 {
				t.Fatalf("expected invalid device to stop flow, got session=%d event=%d commit=%d rollback=%d",
					sessionManager.createCalls, accountRepo.recordLoginEventCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
			}
		})
	}
}

func TestLoginUseCase_SessionCreateFailureStopsBeforeAuditLogHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, uow, _, accountRepo, _, _, _, _, sessionManager, _ := newLoginUseCaseFixture(t)
	sessionManager.createErr = errors.New("redis unavailable")
	seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
	input := loginInput("login@example.com", "redacted-password", loginDeviceID)

	t.Log("Given: session manager fails after device registration")
	t.Log("Input: identifier_type=email password_present=true")
	t.Log("Action: execute login usecase session failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: register_device_calls=%d session_create_calls=%d login_event_calls=%d account_update_calls=%d commit_calls=%d rollback_calls=%d",
		accountRepo.registerDeviceCalls, sessionManager.createCalls, accountRepo.recordLoginEventCalls, accountRepo.updateCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertLoginError(t, err, ErrLoginFailed)
	if accountRepo.registerDeviceCalls != 1 || sessionManager.createCalls != 1 || accountRepo.recordLoginEventCalls != 0 || accountRepo.updateCalls != 0 || uow.tx.commitCalls != 0 || uow.tx.rollbackCalls != 1 {
		t.Fatalf("expected session failure before audit/update/commit, got register=%d session=%d event=%d update=%d commit=%d rollback=%d",
			accountRepo.registerDeviceCalls, sessionManager.createCalls, accountRepo.recordLoginEventCalls, accountRepo.updateCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	}
}

func TestLoginUseCase_CommitFailureReturnsLoginFailedHasStructuredLog(t *testing.T) {
	start := time.Now()
	uc, uow, _, accountRepo, _, _, _, _, _, _ := newLoginUseCaseFixture(t)
	uow.tx = &authRegisterTxStub{commitErr: errors.New("commit failed")}
	seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
	input := loginInput("login@example.com", "redacted-password", loginDeviceID)

	t.Log("Given: all login side effects succeed but transaction commit fails")
	t.Log("Input: identifier_type=email password_present=true")
	t.Log("Action: execute login usecase commit failure path")

	out, err := uc.Execute(context.Background(), input)

	t.Logf("Output: token_present=%t err=%v", out.TokenPair.AccessToken != "", err)
	t.Logf("Mutation: login_event_calls=%d account_update_calls=%d commit_calls=%d rollback_calls=%d", accountRepo.recordLoginEventCalls, accountRepo.updateCalls, uow.tx.commitCalls, uow.tx.rollbackCalls)
	t.Logf("Duration: %s", time.Since(start))

	assertLoginError(t, err, ErrLoginFailed)
	if uow.tx.commitCalls != 1 || uow.tx.rollbackCalls != 1 {
		t.Fatalf("expected commit failure to rollback once, got commit=%d rollback=%d", uow.tx.commitCalls, uow.tx.rollbackCalls)
	}
}

func TestLoginUseCase_NonDevSendsLoginEmailWithDeviceNameHasStructuredLog(t *testing.T) {
	start := time.Now()
	t.Setenv("APP_ENV", "prod")
	uow := &authRegisterUOWStub{}
	hasher := &authHasherStub{verifyResult: true}
	accountRepo := newAuthAccountRepoStub()
	userRepo := &authUserRepoStub{nextID: 501}
	roleRepo := &authUserRoleRepoStub{}
	deviceRepo := newAuthDeviceRepoStub()
	participantRepo := newAuthParticipantRepoStub()
	sessionManager := &authSessionManagerStub{}
	emailService := loginEmailServiceStub{sent: make(chan appEmail.EmailBuilder, 1)}
	factory := &loginMailFactoryCapture{}
	deviceRepo.seed(parseLoginDeviceID(t), "Hiro's Mac")
	seedLoginAccount(accountRepo, "login@example.com", account.Active, 777)
	uc := NewLoginUseCase(uow, hasher, sessionManager, accountRepo, userRepo, roleRepo, deviceRepo, participantRepo, emailService, factory.factory)
	input := loginInput("login@example.com", "redacted-password", loginDeviceID)

	t.Log("Given: production login succeeds and email service is available")
	t.Log("Input: identifier_type=email password_present=true app_env=prod")
	t.Log("Action: execute login usecase non-dev notification path")

	out, err := uc.Execute(context.Background(), input)

	select {
	case <-emailService.sent:
	case <-time.After(time.Second):
		t.Fatal("expected async login email to be sent")
	}

	t.Logf("Output: token_present=%t err=%v email_factory_calls=%d", out.TokenPair.AccessToken != "", err, factory.calls)
	t.Logf("Mutation: login_event_calls=%d account_update_calls=%d email_device_name=%q email_device_id=%s", accountRepo.recordLoginEventCalls, accountRepo.updateCalls, factory.deviceName, factory.deviceID)
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if factory.calls != 1 || factory.deviceName != "Hiro's Mac" || factory.deviceID != loginDeviceID || factory.ip != "203.0.113.10" {
		t.Fatalf("expected login mail factory to receive device name/id/ip, got calls=%d device=%q id=%s ip=%s", factory.calls, factory.deviceName, factory.deviceID, factory.ip)
	}
}

var (
	_ port.SessionManager    = (*authSessionManagerStub)(nil)
	_ device.Repository      = (*authDeviceRepoStub)(nil)
	_ participant.Repository = (*authParticipantRepoStub)(nil)
	_ appEmail.EmailService  = (*loginEmailServiceStub)(nil)
)
