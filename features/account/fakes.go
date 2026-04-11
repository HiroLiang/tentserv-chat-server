package account

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	authPort "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/port"
	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/auth"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	domainSecurity "github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/gofrs/uuid"
)

type Deps struct {
	accountRepo     *bddAccountRepo
	userRepo        *bddUserRepo
	roleRepo        *bddUserRoleRepo
	deviceRepo      *bddDeviceRepo
	participantRepo *bddParticipantRepo
	sessionManager  *bddSessionManager
	store           *bddVerificationStore
	email           *bddEmailService
	registerLimiter *bddRegisterRateLimiter
	loginLimiter    *bddLoginRateLimiter
}

func NewDeps() *Deps {
	deps := &Deps{
		accountRepo:     newBDDAccountRepo(),
		userRepo:        newBDDUserRepo(),
		roleRepo:        &bddUserRoleRepo{},
		deviceRepo:      newBDDDeviceRepo(),
		participantRepo: newBDDParticipantRepo(),
		sessionManager:  newBDDSessionManager(),
		store:           newBDDVerificationStore(),
		email:           &bddEmailService{},
		registerLimiter: &bddRegisterRateLimiter{},
		loginLimiter:    &bddLoginRateLimiter{},
	}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.accountRepo.reset()
	d.userRepo.reset()
	d.roleRepo.reset()
	d.deviceRepo.reset()
	d.participantRepo.reset()
	d.sessionManager.reset()
	d.store.reset()
	d.email.reset()
	d.registerLimiter.reset()
	d.loginLimiter.reset()
}

func (d *Deps) RegisterLimiter() appSecurity.RegisterRateLimiter {
	return d.registerLimiter
}

func (d *Deps) SetRegisterLimitExceeded(exceeded bool) {
	d.registerLimiter.exceeded = exceeded
}

func (d *Deps) SessionManager() authPort.SessionManager {
	return d.sessionManager
}

func (d *Deps) UserRepo() domainuser.Repository {
	return d.userRepo
}

func (d *Deps) LastSessionUserID() shared.UserID {
	d.sessionManager.mu.Lock()
	defer d.sessionManager.mu.Unlock()

	if d.sessionManager.lastCreated == nil {
		return 0
	}
	return d.sessionManager.lastCreated.UserID
}

func (d *Deps) LastSessionDeviceID() shared.DeviceID {
	d.sessionManager.mu.Lock()
	defer d.sessionManager.mu.Unlock()

	if d.sessionManager.lastCreated == nil {
		return shared.DeviceID{}
	}
	return d.sessionManager.lastCreated.DeviceID
}

func (d *Deps) LastAccessToken() auth.AccessToken {
	d.sessionManager.mu.Lock()
	defer d.sessionManager.mu.Unlock()

	if d.sessionManager.lastCreated == nil {
		return ""
	}
	return d.sessionManager.lastCreated.Token.AccessToken
}

func (d *Deps) RevokeLastAccessToken() error {
	token := d.LastAccessToken()
	if token == "" {
		return fmt.Errorf("no access token available to revoke")
	}
	return d.sessionManager.Revoke(context.Background(), token)
}

func (d *Deps) RegisterUseCases(
	uow transaction.UnitOfWork,
	hasher appSecurity.Hasher,
) (*authUseCase.RegisterUseCase, *authUseCase.VerifyEmailUseCase, *authUseCase.ResendVerifyEmailUseCase) {
	registerUseCase := authUseCase.NewRegisterUseCase(
		uow,
		hasher,
		d.accountRepo,
		d.userRepo,
		d.roleRepo,
		d.store,
		d.email,
		func(string, string, string) appEmail.EmailBuilder {
			return bddEmailBuilder{}
		},
	)
	verifyUseCase := authUseCase.NewVerifyEmailUseCase(d.store, d.accountRepo)
	resendVerifyUseCase := authUseCase.NewResendVerifyEmailUseCase(
		d.store,
		d.accountRepo,
		d.email,
		func(string, string, string) appEmail.EmailBuilder {
			return bddEmailBuilder{}
		},
	)
	return registerUseCase, verifyUseCase, resendVerifyUseCase
}

func (d *Deps) LoginUseCases(
	uow transaction.UnitOfWork,
	hasher appSecurity.Hasher,
) (*authUseCase.LoginUseCase, *authUseCase.LogoutUseCase, *authUseCase.GetProfileUseCase) {
	loginUseCase := authUseCase.NewLoginUseCase(
		uow,
		hasher,
		d.loginLimiter,
		d.sessionManager,
		d.accountRepo,
		d.userRepo,
		d.roleRepo,
		d.deviceRepo,
		d.participantRepo,
		d.email,
		func(string, string, string, string, string, time.Time) appEmail.EmailBuilder {
			return bddEmailBuilder{}
		},
	)
	logoutUseCase := authUseCase.NewLogoutUseCase(d.sessionManager)
	profileUseCase := authUseCase.NewGetProfileUseCase(d.accountRepo, d.userRepo)
	return loginUseCase, logoutUseCase, profileUseCase
}

type bddAccountRepo struct {
	mu                    sync.Mutex
	nextID                shared.AccountID
	accountsByID          map[shared.AccountID]*domainaccount.Account
	accountsByEmail       map[shared.EmailAddress]*domainaccount.Account
	accountsByName        map[string]*domainaccount.Account
	createCalls           int
	updateCalls           int
	registerDeviceCalls   int
	recordLoginEventCalls int
	lastUpdated           *domainaccount.Account
	lastRegisteredDevice  *domainaccount.AccountDevice
	lastLoginEvent        *domainaccount.AccountLoginEvent
}

func newBDDAccountRepo() *bddAccountRepo {
	return &bddAccountRepo{}
}

func (r *bddAccountRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID = 100
	r.accountsByID = map[shared.AccountID]*domainaccount.Account{}
	r.accountsByEmail = map[shared.EmailAddress]*domainaccount.Account{}
	r.accountsByName = map[string]*domainaccount.Account{}
	r.createCalls = 0
	r.updateCalls = 0
	r.registerDeviceCalls = 0
	r.recordLoginEventCalls = 0
	r.lastUpdated = nil
	r.lastRegisteredDevice = nil
	r.lastLoginEvent = nil
}

func (r *bddAccountRepo) seed(email shared.EmailAddress, accountName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	acc := &domainaccount.Account{
		ID:          r.nextID,
		PublicID:    uuid.Nil,
		Email:       email,
		AccountName: accountName,
		Password:    "existing-hash",
		Status:      domainaccount.Active,
		UserLimit:   1,
	}
	r.accountsByID[acc.ID] = acc
	r.accountsByEmail[email] = acc
	r.accountsByName[accountName] = acc
}

func (r *bddAccountRepo) seedLogin(email shared.EmailAddress, accountName, passwordHash string, status domainaccount.Status, userIDs ...shared.UserID) *domainaccount.Account {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	acc := &domainaccount.Account{
		ID:          r.nextID,
		PublicID:    uuid.Nil,
		Email:       email,
		AccountName: accountName,
		Password:    passwordHash,
		Status:      status,
		UserLimit:   1,
		UserIDs:     append([]shared.UserID(nil), userIDs...),
	}
	r.accountsByID[acc.ID] = acc
	r.accountsByEmail[email] = acc
	r.accountsByName[accountName] = acc
	return cloneBDDAccount(acc)
}

func (r *bddAccountRepo) FindByID(_ context.Context, id shared.AccountID) (*domainaccount.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	acc, ok := r.accountsByID[id]
	if !ok {
		return nil, domainaccount.ErrAccountNotFound
	}
	return cloneBDDAccount(acc), nil
}

func (r *bddAccountRepo) FindByAccountName(_ context.Context, accountName string) (*domainaccount.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	acc, ok := r.accountsByName[accountName]
	if !ok {
		return nil, domainaccount.ErrAccountNotFound
	}
	return cloneBDDAccount(acc), nil
}

func (r *bddAccountRepo) FindByEmail(_ context.Context, email shared.EmailAddress) (*domainaccount.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	acc, ok := r.accountsByEmail[email]
	if !ok {
		return nil, domainaccount.ErrAccountNotFound
	}
	return cloneBDDAccount(acc), nil
}

func (r *bddAccountRepo) Create(_ context.Context, acc *domainaccount.Account) (shared.AccountID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.accountsByEmail[acc.Email]; ok {
		return 0, domainaccount.ErrEmailExist
	}
	if _, ok := r.accountsByName[acc.AccountName]; ok {
		return 0, domainaccount.ErrAccountExist
	}

	r.nextID++
	created := cloneBDDAccount(acc)
	created.ID = r.nextID
	r.accountsByID[created.ID] = created
	r.accountsByEmail[created.Email] = created
	r.accountsByName[created.AccountName] = created
	r.createCalls++
	return created.ID, nil
}

func (r *bddAccountRepo) Update(_ context.Context, acc *domainaccount.Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	updated := cloneBDDAccount(acc)
	r.accountsByID[updated.ID] = updated
	r.accountsByEmail[updated.Email] = updated
	r.accountsByName[updated.AccountName] = updated
	r.updateCalls++
	r.lastUpdated = updated
	return nil
}

func (r *bddAccountRepo) RegisterDevice(_ context.Context, device *domainaccount.AccountDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registerDeviceCalls++
	copied := *device
	r.lastRegisteredDevice = &copied
	return nil
}

func (r *bddAccountRepo) RecordLoginEvent(_ context.Context, event *domainaccount.AccountLoginEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.recordLoginEventCalls++
	copied := *event
	r.lastLoginEvent = &copied
	return nil
}

func (r *bddAccountRepo) ReplaceDevices(context.Context, shared.AccountID, []domainaccount.AccountDevice) error {
	return nil
}

type bddUserRepo struct {
	mu          sync.Mutex
	nextID      shared.UserID
	usersByID   map[shared.UserID]*domainuser.User
	createCalls int
	updateCalls int
	lastCreated *domainuser.User
	lastUpdated *domainuser.User
}

func newBDDUserRepo() *bddUserRepo {
	return &bddUserRepo{}
}

func (r *bddUserRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID = 500
	r.usersByID = map[shared.UserID]*domainuser.User{}
	r.createCalls = 0
	r.updateCalls = 0
	r.lastCreated = nil
	r.lastUpdated = nil
}

func (r *bddUserRepo) seed(accountID shared.AccountID, userID shared.UserID, name string, roles []role.Code) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u := &domainuser.User{
		ID:        userID,
		AccountID: accountID,
		Name:      name,
		RoleCodes: append([]role.Code(nil), roles...),
	}
	r.usersByID[userID] = u
	if userID > r.nextID {
		r.nextID = userID
	}
}

func (r *bddUserRepo) Create(_ context.Context, u *domainuser.User) (shared.UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	created := *u
	created.ID = r.nextID
	created.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
	r.usersByID[created.ID] = &created
	r.lastCreated = &created
	r.createCalls++
	return created.ID, nil
}

func (r *bddUserRepo) FindByID(_ context.Context, id shared.UserID) (*domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.usersByID[id]
	if !ok {
		return nil, domainuser.ErrUserNotFound
	}
	copied := *u
	copied.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
	return &copied, nil
}

func (r *bddUserRepo) FindByAccountID(_ context.Context, accountID shared.AccountID) (*[]domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users := make([]domainuser.User, 0, len(r.usersByID))
	for _, u := range r.usersByID {
		if u.AccountID != accountID {
			continue
		}
		copied := *u
		copied.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
		users = append(users, copied)
	}
	return &users, nil
}

func (r *bddUserRepo) Update(_ context.Context, u *domainuser.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	updated := *u
	updated.RoleCodes = append([]role.Code(nil), u.RoleCodes...)
	r.usersByID[updated.ID] = &updated
	r.updateCalls++
	r.lastUpdated = &updated
	return nil
}

func (r *bddUserRepo) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (r *bddUserRepo) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (r *bddUserRepo) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

type bddUserRoleRepo struct {
	mu          sync.Mutex
	assignCalls int
	assigned    []role.Code
}

func (r *bddUserRoleRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.assignCalls = 0
	r.assigned = nil
}

func (r *bddUserRoleRepo) FindRolesByUser(context.Context, shared.UserID) ([]*role.Role, error) {
	return nil, nil
}

func (r *bddUserRoleRepo) Exists(context.Context, shared.UserID, role.Code) bool {
	return false
}

func (r *bddUserRoleRepo) Assign(_ context.Context, _ shared.UserID, code role.Code) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.assignCalls++
	r.assigned = append(r.assigned, code)
	return nil
}

func (r *bddUserRoleRepo) Revoke(context.Context, shared.UserID, role.Code) error {
	return nil
}

type bddSessionManager struct {
	mu          sync.Mutex
	nextID      int64
	sessions    map[auth.AccessToken]*auth.Session
	createCalls int
	findCalls   int
	revokeCalls int
	lastCreated *auth.Session
}

func newBDDSessionManager() *bddSessionManager {
	return &bddSessionManager{}
}

func (s *bddSessionManager) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID = 900
	s.sessions = map[auth.AccessToken]*auth.Session{}
	s.createCalls = 0
	s.findCalls = 0
	s.revokeCalls = 0
	s.lastCreated = nil
}

func (s *bddSessionManager) Create(_ context.Context, input auth.CreateSessionInput) (auth.TokenPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	tokenPair := auth.TokenPair{
		AccessToken:  auth.AccessToken(fmt.Sprintf("bdd-access-token-%d", s.nextID)),
		RefreshToken: auth.RefreshToken(fmt.Sprintf("bdd-refresh-token-%d", s.nextID)),
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	session := &auth.Session{
		ID:        fmt.Sprintf("%d", s.nextID),
		AccountID: input.AccountID,
		UserID:    input.UserID,
		DeviceID:  input.DeviceID,
		Token:     tokenPair,
		CreatedAt: time.Now(),
	}
	s.sessions[tokenPair.AccessToken] = session
	s.createCalls++
	s.lastCreated = session
	return tokenPair, nil
}

func (s *bddSessionManager) FindByToken(_ context.Context, token auth.AccessToken) (*auth.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.findCalls++
	session, ok := s.sessions[token]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	copied := *session
	return &copied, nil
}

func (s *bddSessionManager) Refresh(context.Context, auth.RefreshToken) (auth.TokenPair, error) {
	return auth.TokenPair{}, nil
}

func (s *bddSessionManager) Revoke(_ context.Context, token auth.AccessToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.revokeCalls++
	delete(s.sessions, token)
	return nil
}

func (s *bddSessionManager) RevokeAllForUser(context.Context, shared.AccountID) error {
	return nil
}

func (s *bddSessionManager) RevokeAll(context.Context) error {
	return nil
}

func (s *bddSessionManager) SwitchUser(context.Context, auth.AccessToken, shared.UserID) error {
	return nil
}

type bddDeviceRepo struct {
	mu            sync.Mutex
	devicesByID   map[string]*domaindevice.Device
	findByIDCalls int
}

func newBDDDeviceRepo() *bddDeviceRepo {
	return &bddDeviceRepo{}
}

func (r *bddDeviceRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devicesByID = map[string]*domaindevice.Device{}
	r.findByIDCalls = 0
}

func (r *bddDeviceRepo) seed(id shared.DeviceID, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devicesByID[id.String()] = &domaindevice.Device{
		ID:       id,
		Platform: domaindevice.MacOS,
		Name:     name,
	}
}

func (r *bddDeviceRepo) FindByID(_ context.Context, id shared.DeviceID) (*domaindevice.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.findByIDCalls++
	d, ok := r.devicesByID[id.String()]
	if !ok {
		return nil, domaindevice.ErrDeviceNotFound
	}
	copied := *d
	return &copied, nil
}

func (r *bddDeviceRepo) FindAllByAccountID(context.Context, shared.AccountID) ([]*domaindevice.Device, error) {
	return nil, nil
}

func (r *bddDeviceRepo) Create(context.Context, *domaindevice.Device) error {
	return nil
}

func (r *bddDeviceRepo) Update(context.Context, *domaindevice.Device) error {
	return nil
}

func (r *bddDeviceRepo) BindAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

func (r *bddDeviceRepo) DeleteByAccount(context.Context, shared.DeviceID, shared.AccountID) error {
	return nil
}

type bddParticipantRepo struct {
	mu                 sync.Mutex
	nextID             participant.ID
	participantsByUser map[shared.UserID]*participant.Participant
	findByUserCalls    int
	createCalls        int
	lastCreated        *participant.Participant
}

func newBDDParticipantRepo() *bddParticipantRepo {
	return &bddParticipantRepo{}
}

func (r *bddParticipantRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID = 800
	r.participantsByUser = map[shared.UserID]*participant.Participant{}
	r.findByUserCalls = 0
	r.createCalls = 0
	r.lastCreated = nil
}

func (r *bddParticipantRepo) seedUser(userID shared.UserID) *participant.Participant {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	uid := userID
	p := &participant.Participant{
		ID:        r.nextID,
		Type:      participant.UserType,
		UserID:    &uid,
		CreatedAt: time.Now(),
	}
	r.participantsByUser[userID] = p
	return p
}

func (r *bddParticipantRepo) FindByID(context.Context, participant.ID) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *bddParticipantRepo) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.findByUserCalls++
	p, ok := r.participantsByUser[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *bddParticipantRepo) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *bddParticipantRepo) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *bddParticipantRepo) Create(_ context.Context, p *participant.Participant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.createCalls++
	if p.UserID != nil {
		if _, ok := r.participantsByUser[*p.UserID]; ok {
			return participant.ErrAlreadyExists
		}
	}
	r.nextID++
	created := *p
	created.ID = r.nextID
	created.CreatedAt = time.Now()
	r.lastCreated = &created
	p.ID = created.ID
	p.CreatedAt = created.CreatedAt
	if created.UserID != nil {
		r.participantsByUser[*created.UserID] = &created
	}
	return nil
}

type bddVerificationStore struct {
	mu                sync.Mutex
	sessions          map[string]authPort.VerificationSession
	accountTokens     map[int64]string
	storeCalls        int
	getCalls          int
	deleteCalls       int
	lastStoredToken   string
	lastStoredSession authPort.VerificationSession
	deleted           []string
}

func newBDDVerificationStore() *bddVerificationStore {
	return &bddVerificationStore{}
}

func (s *bddVerificationStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = map[string]authPort.VerificationSession{}
	s.accountTokens = map[int64]string{}
	s.storeCalls = 0
	s.getCalls = 0
	s.deleteCalls = 0
	s.lastStoredToken = ""
	s.lastStoredSession = authPort.VerificationSession{}
	s.deleted = nil
}

func (s *bddVerificationStore) Store(_ context.Context, token string, session authPort.VerificationSession, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if prevToken, ok := s.accountTokens[session.AccountID]; ok && prevToken != token {
		delete(s.sessions, prevToken)
	}
	s.storeCalls++
	s.sessions[token] = session
	s.accountTokens[session.AccountID] = token
	s.lastStoredToken = token
	s.lastStoredSession = session
	return nil
}

// expireToken simulates Redis TTL elapse: removes the token from the store without
// incrementing deleteCalls, so mutation assertions can distinguish TTL expiry from
// use-case-initiated Delete calls.
func (s *bddVerificationStore) expireToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if ok {
		delete(s.accountTokens, session.AccountID)
	}
	delete(s.sessions, token)
}

func (s *bddVerificationStore) Get(_ context.Context, token string) (authPort.VerificationSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.getCalls++
	session, ok := s.sessions[token]
	return session, ok, nil
}

func (s *bddVerificationStore) FindTokenByAccountID(_ context.Context, accountID int64) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.accountTokens[accountID]
	return token, ok, nil
}

func (s *bddVerificationStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteCalls++
	s.deleted = append(s.deleted, token)
	session, ok := s.sessions[token]
	if ok {
		delete(s.accountTokens, session.AccountID)
	}
	delete(s.sessions, token)
	return nil
}

type bddEmailService struct {
	mu        sync.Mutex
	sendCalls int
}

func (s *bddEmailService) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sendCalls = 0
}

func (s *bddEmailService) Send(context.Context, appEmail.EmailBuilder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sendCalls++
	return nil
}

type bddEmailBuilder struct{}

func (bddEmailBuilder) BuildEmail(context.Context) (*shared.Email, error) {
	return &shared.Email{}, nil
}

func cloneBDDAccount(acc *domainaccount.Account) *domainaccount.Account {
	cloned := *acc
	cloned.UserIDs = append([]shared.UserID(nil), acc.UserIDs...)
	cloned.Devices = append([]domainaccount.AccountDevice(nil), acc.Devices...)
	return &cloned
}

type bddRegisterRateLimiter struct {
	mu       sync.Mutex
	exceeded bool
	calls    int
}

func (l *bddRegisterRateLimiter) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.exceeded = false
	l.calls = 0
}

func (l *bddRegisterRateLimiter) CheckRegisterAttempt(_ context.Context, _ string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.exceeded {
		return errors.New("registration rate limit exceeded")
	}
	return nil
}

type bddLoginRateLimiter struct {
	mu           sync.Mutex
	failureCount map[string]int64
	checkCalls   int
	recordCalls  int
	releaseCalls int
}

func (l *bddLoginRateLimiter) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.failureCount = map[string]int64{}
	l.checkCalls = 0
	l.recordCalls = 0
	l.releaseCalls = 0
}

func (l *bddLoginRateLimiter) CheckLoginAttempt(_ context.Context, _, identifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.checkCalls++
	if l.failureCount[identifier] >= 5 {
		return domainSecurity.ErrRateLimitExceeded
	}
	return nil
}

func (l *bddLoginRateLimiter) RecordLoginAttempt(_ context.Context, _, identifier string, success bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.recordCalls++
	if success {
		delete(l.failureCount, identifier)
		return nil
	}
	l.failureCount[identifier]++
	return nil
}

func (l *bddLoginRateLimiter) ReleaseLock(_ context.Context, identifier string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.releaseCalls++
	delete(l.failureCount, identifier)
	return nil
}

func (l *bddLoginRateLimiter) count(identifier string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.failureCount[identifier]
}

var (
	_ domainaccount.Repository        = (*bddAccountRepo)(nil)
	_ domainuser.Repository           = (*bddUserRepo)(nil)
	_ userrole.Repository             = (*bddUserRoleRepo)(nil)
	_ authPort.SessionManager         = (*bddSessionManager)(nil)
	_ domaindevice.Repository         = (*bddDeviceRepo)(nil)
	_ participant.Repository          = (*bddParticipantRepo)(nil)
	_ appEmail.EmailService           = (*bddEmailService)(nil)
	_ appEmail.EmailBuilder           = (*bddEmailBuilder)(nil)
	_ appSecurity.RegisterRateLimiter = (*bddRegisterRateLimiter)(nil)
	_ appSecurity.LoginRateLimiter    = (*bddLoginRateLimiter)(nil)
)
