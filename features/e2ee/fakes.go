package e2ee

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	e2eeUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	appCrypto "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/crypto"
	appPush "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/useridentitykey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/userotpprekey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/usersignedprekey"
)

type Deps struct {
	identityRepo *bddIdentityKeyRepo
	signedRepo   *bddSignedPreKeyRepo
	otpRepo      *bddOTPPreKeyRepo
	verifier     *bddKeyVerifier
	dispatcher   *bddDispatcher
	SKR          *SKRDeps
}

type UseCases struct {
	UploadIdentityKey  *e2eeUseCase.UploadIdentityKeyUseCase
	UploadSignedPreKey *e2eeUseCase.UploadSignedPreKeyUseCase
	UploadOTPPreKeys   *e2eeUseCase.UploadOTPPreKeysUseCase
	CountOTPPreKeys    *e2eeUseCase.CountOTPPreKeysUseCase
	GetKeyBundle       *e2eeUseCase.GetKeyBundleUseCase
	CheckKeyStatus     *e2eeUseCase.CheckKeyStatusUseCase
	GetKeyPolicy       *e2eeUseCase.GetKeyPolicyUseCase
}

func NewDeps() *Deps {
	deps := &Deps{
		identityRepo: newBDDIdentityKeyRepo(),
		signedRepo:   newBDDSignedPreKeyRepo(),
		otpRepo:      newBDDOTPPreKeyRepo(),
		verifier:     &bddKeyVerifier{},
		dispatcher:   &bddDispatcher{},
		SKR:          NewSKRDeps(),
	}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.identityRepo.reset()
	d.signedRepo.reset()
	d.otpRepo.reset()
	d.dispatcher.reset()
	d.SKR.Reset()
}

func (d *Deps) RegisterUseCases() UseCases {
	return UseCases{
		UploadIdentityKey:  e2eeUseCase.NewUploadIdentityKeyUseCase(d.identityRepo, d.verifier),
		UploadSignedPreKey: e2eeUseCase.NewUploadSignedPreKeyUseCase(d.identityRepo, d.signedRepo, d.verifier),
		UploadOTPPreKeys:   e2eeUseCase.NewUploadOTPPreKeysUseCase(d.otpRepo),
		CountOTPPreKeys:    e2eeUseCase.NewCountOTPPreKeysUseCase(d.otpRepo),
		GetKeyBundle:       e2eeUseCase.NewGetKeyBundleUseCase(d.identityRepo, d.signedRepo, d.otpRepo, d.dispatcher),
		CheckKeyStatus:     e2eeUseCase.NewCheckKeyStatusUseCase(d.identityRepo, d.signedRepo, d.otpRepo),
		GetKeyPolicy:       e2eeUseCase.NewGetKeyPolicyUseCase(),
	}
}

func (d *Deps) RepositoryCounts() (identity, signed, otp int) {
	return d.identityRepo.count(), d.signedRepo.countActive(), d.otpRepo.countAll()
}

func (d *Deps) OTPConsumeCalls() int {
	return d.otpRepo.consumeCalls()
}

func (d *Deps) OTPCount(userID user.ID, deviceID device.ID) int {
	count, _ := d.otpRepo.CountAvailable(context.Background(), userID, deviceID)
	return count
}

func (d *Deps) LastDispatch() (userID shared.UserID, msgType string, payload []byte, calls int) {
	return d.dispatcher.last()
}

type bddIdentityKeyRepo struct {
	mu     sync.Mutex
	byKey  map[string]*useridentitykey.UserIdentityKey
	upsert int
}

func newBDDIdentityKeyRepo() *bddIdentityKeyRepo {
	return &bddIdentityKeyRepo{}
}

func (r *bddIdentityKeyRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey = map[string]*useridentitykey.UserIdentityKey{}
	r.upsert = 0
}

func (r *bddIdentityKeyRepo) FindByUserAndDevice(_ context.Context, userID user.ID, deviceID device.ID) (*useridentitykey.UserIdentityKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, ok := r.byKey[e2eeRepoKey(userID, deviceID)]
	if !ok {
		return nil, useridentitykey.ErrNotFound
	}
	return cloneIdentityKey(key), nil
}

func (r *bddIdentityKeyRepo) FindByUser(_ context.Context, userID user.ID) ([]*useridentitykey.UserIdentityKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := make([]*useridentitykey.UserIdentityKey, 0)
	for _, key := range r.byKey {
		if key.UserID == userID {
			keys = append(keys, cloneIdentityKey(key))
		}
	}
	return keys, nil
}

func (r *bddIdentityKeyRepo) Upsert(_ context.Context, key *useridentitykey.UserIdentityKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey[e2eeRepoKey(key.UserID, key.DeviceID)] = cloneIdentityKey(key)
	r.upsert++
	return nil
}

func (r *bddIdentityKeyRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byKey)
}

type bddSignedPreKeyRepo struct {
	mu     sync.Mutex
	byKey  map[string]*usersignedprekey.UserSignedPreKey
	adds   int
	active int
}

func newBDDSignedPreKeyRepo() *bddSignedPreKeyRepo {
	return &bddSignedPreKeyRepo{}
}

func (r *bddSignedPreKeyRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey = map[string]*usersignedprekey.UserSignedPreKey{}
	r.adds = 0
	r.active = 0
}

func (r *bddSignedPreKeyRepo) FindActive(_ context.Context, userID user.ID, deviceID device.ID) (*usersignedprekey.UserSignedPreKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, key := range r.byKey {
		if key.UserID == userID && key.DeviceID == deviceID && key.IsActive {
			return cloneSignedPreKey(key), nil
		}
	}
	return nil, usersignedprekey.ErrNotFound
}

func (r *bddSignedPreKeyRepo) FindByKeyID(_ context.Context, userID user.ID, deviceID device.ID, keyID usersignedprekey.KeyID) (*usersignedprekey.UserSignedPreKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key, ok := r.byKey[e2eeSignedRepoKey(userID, deviceID, keyID)]
	if !ok {
		return nil, usersignedprekey.ErrNotFound
	}
	return cloneSignedPreKey(key), nil
}

func (r *bddSignedPreKeyRepo) Add(_ context.Context, key *usersignedprekey.UserSignedPreKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey[e2eeSignedRepoKey(key.UserID, key.DeviceID, key.KeyID)] = cloneSignedPreKey(key)
	r.adds++
	return nil
}

func (r *bddSignedPreKeyRepo) DeactivateAll(_ context.Context, userID user.ID, deviceID device.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, key := range r.byKey {
		if key.UserID == userID && key.DeviceID == deviceID {
			key.IsActive = false
		}
	}
	return nil
}

func (r *bddSignedPreKeyRepo) countActive() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, key := range r.byKey {
		if key.IsActive {
			count++
		}
	}
	return count
}

type bddOTPPreKeyRepo struct {
	mu       sync.Mutex
	byKey    map[string]*userotpprekey.UserOTPPreKey
	adds     int
	consumes int
}

func newBDDOTPPreKeyRepo() *bddOTPPreKeyRepo {
	return &bddOTPPreKeyRepo{}
}

func (r *bddOTPPreKeyRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey = map[string]*userotpprekey.UserOTPPreKey{}
	r.adds = 0
	r.consumes = 0
}

func (r *bddOTPPreKeyRepo) ConsumeOne(_ context.Context, userID user.ID, deviceID device.ID) (*userotpprekey.UserOTPPreKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.consumes++
	var selectedRepoKey string
	var selected *userotpprekey.UserOTPPreKey
	for key, otp := range r.byKey {
		if otp.UserID == userID && otp.DeviceID == deviceID {
			if selected == nil || otp.KeyID < selected.KeyID {
				selectedRepoKey = key
				selected = otp
			}
		}
	}
	if selected != nil {
		delete(r.byKey, selectedRepoKey)
		return cloneOTPPreKey(selected), nil
	}
	return nil, userotpprekey.ErrPoolEmpty
}

func (r *bddOTPPreKeyRepo) AddBatch(_ context.Context, keys []*userotpprekey.UserOTPPreKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, key := range keys {
		r.byKey[e2eeOTPRepoKey(key.UserID, key.DeviceID, key.KeyID)] = cloneOTPPreKey(key)
	}
	r.adds++
	return nil
}

func (r *bddOTPPreKeyRepo) CountAvailable(_ context.Context, userID user.ID, deviceID device.ID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, otp := range r.byKey {
		if otp.UserID == userID && otp.DeviceID == deviceID {
			count++
		}
	}
	return count, nil
}

func (r *bddOTPPreKeyRepo) countAll() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byKey)
}

func (r *bddOTPPreKeyRepo) consumeCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.consumes
}

type bddKeyVerifier struct{}

type bddDispatchRecord struct {
	userID  shared.UserID
	msgType string
	payload []byte
}

type bddDispatcher struct {
	mu      sync.Mutex
	records []bddDispatchRecord
}

func (d *bddDispatcher) reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records = nil
}

func (d *bddDispatcher) Dispatch(_ context.Context, userID shared.UserID, msgType string, payload []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records = append(d.records, bddDispatchRecord{
		userID:  userID,
		msgType: msgType,
		payload: append([]byte(nil), payload...),
	})
	return nil
}

func (d *bddDispatcher) last() (userID shared.UserID, msgType string, payload []byte, calls int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.records) == 0 {
		return 0, "", nil, 0
	}
	record := d.records[len(d.records)-1]
	return record.userID, record.msgType, append([]byte(nil), record.payload...), len(d.records)
}

func (bddKeyVerifier) VerifySignedPreKey(_, _, _ []byte) bool {
	return true
}

func (bddKeyVerifier) FingerprintKey(publicKey []byte) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:])
}

func e2eeRepoKey(userID user.ID, deviceID device.ID) string {
	return fmt.Sprintf("%d:%s", userID, deviceID.String())
}

func e2eeSignedRepoKey(userID user.ID, deviceID device.ID, keyID usersignedprekey.KeyID) string {
	return fmt.Sprintf("%s:%d", e2eeRepoKey(userID, deviceID), keyID)
}

func e2eeOTPRepoKey(userID user.ID, deviceID device.ID, keyID userotpprekey.KeyID) string {
	return fmt.Sprintf("%s:%d", e2eeRepoKey(userID, deviceID), keyID)
}

func cloneIdentityKey(key *useridentitykey.UserIdentityKey) *useridentitykey.UserIdentityKey {
	cloned := *key
	return &cloned
}

func cloneSignedPreKey(key *usersignedprekey.UserSignedPreKey) *usersignedprekey.UserSignedPreKey {
	cloned := *key
	return &cloned
}

func cloneOTPPreKey(key *userotpprekey.UserOTPPreKey) *userotpprekey.UserOTPPreKey {
	cloned := *key
	return &cloned
}

var (
	_ useridentitykey.Repository  = (*bddIdentityKeyRepo)(nil)
	_ usersignedprekey.Repository = (*bddSignedPreKeyRepo)(nil)
	_ userotpprekey.Repository    = (*bddOTPPreKeyRepo)(nil)
	_ appCrypto.KeyVerifier       = (*bddKeyVerifier)(nil)
	_ appPush.Dispatcher          = (*bddDispatcher)(nil)
)
