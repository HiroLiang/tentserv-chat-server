package account

import (
	"context"
	"sync"
	"time"

	authUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/auth/usecase"
	appEmail "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userrole"
	"github.com/gofrs/uuid"
)

type Deps struct {
	accountRepo *bddAccountRepo
	userRepo    *bddUserRepo
	roleRepo    *bddUserRoleRepo
	store       *bddVerificationStore
	email       *bddEmailService
}

func NewDeps() *Deps {
	deps := &Deps{
		accountRepo: newBDDAccountRepo(),
		userRepo:    &bddUserRepo{},
		roleRepo:    &bddUserRoleRepo{},
		store:       newBDDVerificationStore(),
		email:       &bddEmailService{},
	}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.accountRepo.reset()
	d.userRepo.reset()
	d.roleRepo.reset()
	d.store.reset()
	d.email.reset()
}

func (d *Deps) RegisterUseCases(
	uow transaction.UnitOfWork,
	hasher appSecurity.Hasher,
) (*authUseCase.RegisterUseCase, *authUseCase.VerifyEmailUseCase) {
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
	return registerUseCase, verifyUseCase
}

type bddAccountRepo struct {
	mu              sync.Mutex
	nextID          shared.AccountID
	accountsByID    map[shared.AccountID]*domainaccount.Account
	accountsByEmail map[shared.EmailAddress]*domainaccount.Account
	accountsByName  map[string]*domainaccount.Account
	createCalls     int
	updateCalls     int
	lastUpdated     *domainaccount.Account
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
	r.lastUpdated = nil
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

func (r *bddAccountRepo) RegisterDevice(context.Context, *domainaccount.AccountDevice) error {
	return nil
}

func (r *bddAccountRepo) ReplaceDevices(context.Context, shared.AccountID, []domainaccount.AccountDevice) error {
	return nil
}

type bddUserRepo struct {
	mu          sync.Mutex
	nextID      shared.UserID
	createCalls int
	lastCreated *domainuser.User
}

func (r *bddUserRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID = 500
	r.createCalls = 0
	r.lastCreated = nil
}

func (r *bddUserRepo) Create(_ context.Context, u *domainuser.User) (shared.UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	created := *u
	created.ID = r.nextID
	r.lastCreated = &created
	r.createCalls++
	return created.ID, nil
}

func (r *bddUserRepo) FindByID(context.Context, shared.UserID) (*domainuser.User, error) {
	return nil, domainuser.ErrUserNotFound
}

func (r *bddUserRepo) FindByAccountID(context.Context, shared.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (r *bddUserRepo) Update(context.Context, *domainuser.User) error {
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

type bddVerificationStore struct {
	mu              sync.Mutex
	tokens          map[string]int64
	storeCalls      int
	getCalls        int
	deleteCalls     int
	lastStoredToken string
	deleted         []string
}

func newBDDVerificationStore() *bddVerificationStore {
	return &bddVerificationStore{}
}

func (s *bddVerificationStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens = map[string]int64{}
	s.storeCalls = 0
	s.getCalls = 0
	s.deleteCalls = 0
	s.lastStoredToken = ""
	s.deleted = nil
}

func (s *bddVerificationStore) Store(_ context.Context, token string, accountID int64, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.storeCalls++
	s.tokens[token] = accountID
	s.lastStoredToken = token
	return nil
}

func (s *bddVerificationStore) Get(_ context.Context, token string) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.getCalls++
	id, ok := s.tokens[token]
	return id, ok, nil
}

func (s *bddVerificationStore) Delete(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteCalls++
	s.deleted = append(s.deleted, token)
	delete(s.tokens, token)
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

var (
	_ domainaccount.Repository = (*bddAccountRepo)(nil)
	_ domainuser.Repository    = (*bddUserRepo)(nil)
	_ userrole.Repository      = (*bddUserRoleRepo)(nil)
	_ appEmail.EmailService    = (*bddEmailService)(nil)
	_ appEmail.EmailBuilder    = (*bddEmailBuilder)(nil)
)
