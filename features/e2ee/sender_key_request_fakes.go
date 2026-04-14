package e2ee

import (
	"context"
	"fmt"
	"sync"
	"time"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	e2eeUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	domainaccount "github.com/HiroLiang/tentserv-chat-server/internal/domain/account"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	domaindevice "github.com/HiroLiang/tentserv-chat-server/internal/domain/device"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysyncdistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

// noBroadcaster is a no-op broadcaster for BDD tests.
type noBroadcaster struct{}

func (n *noBroadcaster) SendToUser(string, []byte) {}

var _ e2eePort.Broadcaster = (*noBroadcaster)(nil)

// recordingBroadcaster captures SendToUser calls for assertion in BDD steps.
type recordingBroadcaster struct {
	mu       sync.Mutex
	messages []broadcastRecord
}

type broadcastRecord struct {
	UserID  string
	Payload []byte
}

func (r *recordingBroadcaster) SendToUser(userID string, payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]byte, len(payload))
	copy(copied, payload)
	r.messages = append(r.messages, broadcastRecord{UserID: userID, Payload: copied})
}

func (r *recordingBroadcaster) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = nil
}

func (r *recordingBroadcaster) Messages() []broadcastRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]broadcastRecord, len(r.messages))
	copy(out, r.messages)
	return out
}

var _ e2eePort.Broadcaster = (*recordingBroadcaster)(nil)

// SKRDeps holds the in-memory repositories needed for sender key request BDD scenarios.
type SKRDeps struct {
	mu               sync.Mutex
	accountRepo      *skrAccountRepo
	userRepo         *skrUserRepo
	participantRepo  *skrParticipantRepo
	chatMemberRepo   *skrChatMemberRepo
	skrRepo          *skrSenderKeyRequestRepo
	mskRepo          *skrMemberSenderKeyRepo
	distributionRepo *skrSenderKeyDistributionRepo
	receiptRepo      *skrSenderKeyReceiptRepo
	friendshipRepo   *skrFriendshipRepo
	deviceRepo       *skrDeviceRepo
	selfSyncRepo     *skrSelfSenderKeySyncRepo
	selfSyncCopyRepo *skrSelfSenderKeySyncDistributionRepo
	broadcaster      *recordingBroadcaster
}

func NewSKRDeps() *SKRDeps {
	return &SKRDeps{
		accountRepo:      &skrAccountRepo{byID: map[sharedDomain.AccountID]*domainaccount.Account{}},
		userRepo:         &skrUserRepo{byID: map[sharedDomain.UserID]*domainuser.User{}},
		participantRepo:  &skrParticipantRepo{byUserID: map[sharedDomain.UserID]*participant.Participant{}, byID: map[participant.ID]*participant.Participant{}},
		chatMemberRepo:   &skrChatMemberRepo{byID: map[chatmember.ID]*chatmember.ChatMember{}, byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{}},
		skrRepo:          &skrSenderKeyRequestRepo{records: map[string]*senderkeyrequest.SenderKeyRequest{}},
		mskRepo:          &skrMemberSenderKeyRepo{keys: map[string]*membersenderkey.MemberSenderKey{}},
		distributionRepo: &skrSenderKeyDistributionRepo{byPair: map[string]*senderkeydistribution.SenderKeyDistribution{}, byID: map[senderkeydistribution.ID]*senderkeydistribution.SenderKeyDistribution{}},
		receiptRepo:      &skrSenderKeyReceiptRepo{records: map[string]*senderkeyreceipt.SenderKeyReceipt{}},
		friendshipRepo:   &skrFriendshipRepo{},
		deviceRepo:       &skrDeviceRepo{byID: map[sharedDomain.DeviceID]*domaindevice.Device{}},
		selfSyncRepo:     &skrSelfSenderKeySyncRepo{byParticipantID: map[participant.ID]*selfsenderkeysync.SelfSenderKeySync{}},
		selfSyncCopyRepo: &skrSelfSenderKeySyncDistributionRepo{rows: map[string]map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{}},
		broadcaster:      &recordingBroadcaster{},
	}
}

func (d *SKRDeps) Reset() {
	d.accountRepo.reset()
	d.userRepo.reset()
	d.participantRepo.reset()
	d.chatMemberRepo.reset()
	d.skrRepo.reset()
	d.mskRepo.reset()
	d.distributionRepo.reset()
	d.receiptRepo.reset()
	d.friendshipRepo.reset()
	d.deviceRepo.reset()
	d.selfSyncRepo.reset()
	d.selfSyncCopyRepo.reset()
	d.broadcaster.reset()
}

func (d *SKRDeps) BroadcastMessages() []broadcastRecord {
	return d.broadcaster.Messages()
}

func (d *SKRDeps) RegisterCreateSenderKeyRequestUseCase() *e2eeUseCase.CreateSenderKeyRequestUseCase {
	return e2eeUseCase.NewCreateSenderKeyRequestUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.skrRepo,
		d.mskRepo,
		d.distributionRepo,
		d.receiptRepo,
		d.selfSyncRepo,
		d.friendshipRepo,
		&noBroadcaster{},
	)
}

func (d *SKRDeps) RegisterUploadSenderKeyUseCase() *e2eeUseCase.UploadSenderKeyUseCase {
	return e2eeUseCase.NewUploadSenderKeyUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.accountRepo,
		d.userRepo,
		d.mskRepo,
		d.distributionRepo,
		d.skrRepo,
		d.selfSyncRepo,
		d.friendshipRepo,
		&noBroadcaster{},
	)
}

func (d *SKRDeps) RegisterGetSenderKeyDistributionStatusUseCase() *e2eeUseCase.GetSenderKeyDistributionStatusUseCase {
	return e2eeUseCase.NewGetSenderKeyDistributionStatusUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.accountRepo,
		d.userRepo,
		d.mskRepo,
		d.distributionRepo,
		d.receiptRepo,
	)
}

func (d *SKRDeps) RegisterGetSenderKeysUseCase() *e2eeUseCase.GetSenderKeysUseCase {
	return e2eeUseCase.NewGetSenderKeysUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.mskRepo,
		d.receiptRepo,
	)
}

func (d *SKRDeps) RegisterGetPendingSenderKeyDistributionsUseCase() *e2eeUseCase.GetPendingSenderKeyDistributionsUseCase {
	return e2eeUseCase.NewGetPendingSenderKeyDistributionsUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.distributionRepo,
	)
}

func (d *SKRDeps) RegisterConsumeSenderKeyDistributionUseCase() *e2eeUseCase.ConsumeSenderKeyDistributionUseCase {
	return e2eeUseCase.NewConsumeSenderKeyDistributionUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.distributionRepo,
		d.receiptRepo,
		d.skrRepo,
		d.broadcaster,
	)
}

func (d *SKRDeps) RegisterGetSelfSenderKeySyncUseCase() *e2eeUseCase.GetSelfSenderKeySyncUseCase {
	return e2eeUseCase.NewGetSelfSenderKeySyncUseCase(
		d.participantRepo,
		d.selfSyncRepo,
		d.accountRepo,
		d.deviceRepo,
	)
}

func (d *SKRDeps) RegisterUploadSelfSenderKeySyncDistributionsUseCase() *e2eeUseCase.UploadSelfSenderKeySyncDistributionsUseCase {
	return e2eeUseCase.NewUploadSelfSenderKeySyncDistributionsUseCase(
		d.participantRepo,
		d.selfSyncRepo,
		d.selfSyncCopyRepo,
	)
}

func (d *SKRDeps) RegisterGetPendingSelfSenderKeySyncDistributionsUseCase() *e2eeUseCase.GetPendingSelfSenderKeySyncDistributionsUseCase {
	return e2eeUseCase.NewGetPendingSelfSenderKeySyncDistributionsUseCase(
		d.participantRepo,
		d.selfSyncRepo,
		d.selfSyncCopyRepo,
	)
}

func (d *SKRDeps) RegisterConsumeSelfSenderKeySyncDistributionUseCase() *e2eeUseCase.ConsumeSelfSenderKeySyncDistributionUseCase {
	return e2eeUseCase.NewConsumeSelfSenderKeySyncDistributionUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.selfSyncRepo,
		d.selfSyncCopyRepo,
		d.receiptRepo,
	)
}

func (d *SKRDeps) RegisterSelfSenderKeySyncMutationUseCase() *e2eeUseCase.SelfSenderKeySyncMutationUseCase {
	return e2eeUseCase.NewSelfSenderKeySyncMutationUseCase(
		d.participantRepo,
		d.selfSyncRepo,
		d.selfSyncCopyRepo,
		d.accountRepo,
		d.deviceRepo,
		d.broadcaster,
	)
}

// SeedParticipant creates a participant for a user without adding room membership.
func (d *SKRDeps) SeedParticipant(userID sharedDomain.UserID, participantID participant.ID) {
	d.ensureUserAccount(userID)
	p := &participant.Participant{ID: participantID, Type: participant.UserType, UserID: &userID, CreatedAt: time.Now()}
	d.participantRepo.byUserID[userID] = p
	d.participantRepo.byID[participantID] = p
}

// SeedRoomMember creates only the room membership row for an existing participant.
func (d *SKRDeps) SeedRoomMember(participantID participant.ID, memberID chatmember.ID, roomID chatroom.ID) {
	m := &chatmember.ChatMember{ID: memberID, RoomID: roomID, ParticipantID: participantID}
	d.chatMemberRepo.byID[memberID] = m
	if d.chatMemberRepo.byRoomAndParticipant[roomID] == nil {
		d.chatMemberRepo.byRoomAndParticipant[roomID] = map[participant.ID]*chatmember.ChatMember{}
	}
	d.chatMemberRepo.byRoomAndParticipant[roomID][participantID] = m
}

// SeedMember creates a participant + chat member for a user in a room.
func (d *SKRDeps) SeedMember(userID sharedDomain.UserID, participantID participant.ID, memberID chatmember.ID, roomID chatroom.ID) {
	d.SeedParticipant(userID, participantID)
	d.SeedRoomMember(participantID, memberID, roomID)
}

// SeedProviderKey stores a sender key for the given member (provider already has a key).
func (d *SKRDeps) SeedProviderKey(memberID chatmember.ID) {
	d.SeedProviderKeyVersion(memberID, 1)
}

func (d *SKRDeps) SeedProviderKeyVersion(memberID chatmember.ID, version int64) {
	d.SeedProviderKeyVersionForDevice(memberID, sharedDomain.DeviceID{}, version)
}

func (d *SKRDeps) SeedProviderKeyVersionForDevice(memberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, version int64) {
	d.mskRepo.keys[d.mskRepo.key(memberID, senderDeviceID)] = &membersenderkey.MemberSenderKey{
		ChatMemberID:     memberID,
		SenderDeviceID:   senderDeviceID,
		SenderKeyVersion: version,
		ChainID:          membersenderkey.ChainID(version),
	}
}

func (d *SKRDeps) SeedDistribution(roomID int64, senderMemberID, receiverMemberID chatmember.ID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	return d.distributionRepo.seed(roomID, senderMemberID, sharedDomain.DeviceID{}, receiverMemberID, sharedDomain.DeviceID{}, version, status)
}

func (d *SKRDeps) SeedDistributionForDevice(roomID int64, senderMemberID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	return d.SeedDistributionForPair(roomID, senderMemberID, sharedDomain.DeviceID{}, receiverMemberID, receiverDeviceID, version, status)
}

func (d *SKRDeps) SeedDistributionForPair(roomID int64, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	return d.distributionRepo.seed(roomID, senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID, version, status)
}

func (d *SKRDeps) FindDistribution(senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, bool) {
	return d.distributionRepo.findLatest(senderMemberID, receiverMemberID)
}

func (d *SKRDeps) FindDistributionByID(id senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, bool) {
	return d.distributionRepo.findByID(id)
}

func (d *SKRDeps) SeedBlockedFriendship(userID, friendID sharedDomain.UserID) {
	d.friendshipRepo.seedBlocked(userID, friendID)
}

// FindSKRRequest returns the sender key request row (if any) for the given pair.
func (d *SKRDeps) FindSKRRequest(requesterMemberID, providerMemberID chatmember.ID) (*senderkeyrequest.SenderKeyRequest, bool) {
	return d.skrRepo.find(requesterMemberID, providerMemberID)
}

func (d *SKRDeps) PendingSKRCount(requesterMemberID, providerMemberID chatmember.ID) int {
	return d.skrRepo.countPending(requesterMemberID, providerMemberID)
}

func (d *SKRDeps) FulfilledSKRCount(requesterMemberID, providerMemberID chatmember.ID) int {
	return d.skrRepo.countFulfilled(requesterMemberID, providerMemberID)
}

func (d *SKRDeps) SeedPendingSKRRequest(requesterMemberID, providerMemberID chatmember.ID) {
	_ = d.skrRepo.Upsert(context.Background(), &senderkeyrequest.SenderKeyRequest{
		RequesterMemberID: requesterMemberID,
		ProviderMemberID:  providerMemberID,
	})
}

func (d *SKRDeps) SeedPendingSKRRequestForDevice(requesterMemberID, providerMemberID chatmember.ID, requesterDeviceID sharedDomain.DeviceID) {
	_ = d.skrRepo.Upsert(context.Background(), &senderkeyrequest.SenderKeyRequest{
		RequesterMemberID: requesterMemberID,
		ProviderMemberID:  providerMemberID,
		RequesterDeviceID: requesterDeviceID,
	})
}

func (d *SKRDeps) SeedPendingSKRRequestForPair(requesterMemberID chatmember.ID, requesterDeviceID sharedDomain.DeviceID, providerMemberID chatmember.ID, providerDeviceID sharedDomain.DeviceID) {
	_ = d.skrRepo.Upsert(context.Background(), &senderkeyrequest.SenderKeyRequest{
		RequesterMemberID: requesterMemberID,
		ProviderMemberID:  providerMemberID,
		RequesterDeviceID: requesterDeviceID,
		ProviderDeviceID:  providerDeviceID,
	})
}

func (d *SKRDeps) SeedSelfSenderKeySync(syncState *selfsenderkeysync.SelfSenderKeySync) {
	d.selfSyncRepo.seed(syncState)
}

func (d *SKRDeps) SetSelfSyncSnapshotLookupFailure(enabled bool) {
	d.accountRepo.setFailFindByID(enabled)
}

func (d *SKRDeps) SeedAccountBinding(accountID sharedDomain.AccountID, deviceID sharedDomain.DeviceID, name string, platform domaindevice.Platform, status domainaccount.DeviceStatus) {
	d.accountRepo.seedDevice(accountID, deviceID, status)
	d.deviceRepo.seed(&domaindevice.Device{
		ID:        deviceID,
		Name:      name,
		Platform:  platform,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

func (d *SKRDeps) ensureUserAccount(userID sharedDomain.UserID) {
	d.userRepo.mu.Lock()
	if _, ok := d.userRepo.byID[userID]; !ok {
		d.userRepo.byID[userID] = &domainuser.User{
			ID:        userID,
			AccountID: sharedDomain.AccountID(userID),
			Name:      fmt.Sprintf("user-%d", userID),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
	d.userRepo.mu.Unlock()

	d.accountRepo.mu.Lock()
	defer d.accountRepo.mu.Unlock()
	accountID := sharedDomain.AccountID(userID)
	if _, ok := d.accountRepo.byID[accountID]; ok {
		return
	}
	d.accountRepo.byID[accountID] = &domainaccount.Account{
		ID:        accountID,
		Status:    domainaccount.Active,
		UserIDs:   []sharedDomain.UserID{userID},
		Devices:   []domainaccount.AccountDevice{defaultReadyAccountDevice(accountID, userID)},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func defaultReadyAccountDevice(accountID sharedDomain.AccountID, userID sharedDomain.UserID) domainaccount.AccountDevice {
	deviceID, _ := sharedDomain.ParseDeviceID(fmt.Sprintf("00000000-0000-0000-0000-%012d", int64(userID)))
	return domainaccount.AccountDevice{
		AccountID:  accountID,
		DeviceID:   deviceID,
		Status:     domainaccount.DeviceStatusReady,
		LastSeenAt: time.Now(),
	}
}

// ─── account repo ─────────────────────────────────────────────────────────────

type skrAccountRepo struct {
	mu           sync.Mutex
	byID         map[sharedDomain.AccountID]*domainaccount.Account
	failFindByID bool
}

func (r *skrAccountRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = map[sharedDomain.AccountID]*domainaccount.Account{}
	r.failFindByID = false
}

func (r *skrAccountRepo) FindByID(_ context.Context, id sharedDomain.AccountID) (*domainaccount.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failFindByID {
		return nil, domainaccount.ErrAccountNotFound
	}
	acc, ok := r.byID[id]
	if !ok {
		return nil, domainaccount.ErrAccountNotFound
	}
	copied := *acc
	copied.Devices = append([]domainaccount.AccountDevice(nil), acc.Devices...)
	copied.UserIDs = append([]sharedDomain.UserID(nil), acc.UserIDs...)
	return &copied, nil
}

func (r *skrAccountRepo) FindByAccountName(context.Context, string) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (r *skrAccountRepo) FindByEmail(context.Context, sharedDomain.EmailAddress) (*domainaccount.Account, error) {
	return nil, domainaccount.ErrAccountNotFound
}

func (r *skrAccountRepo) Create(context.Context, *domainaccount.Account) (sharedDomain.AccountID, error) {
	return 0, nil
}

func (r *skrAccountRepo) Update(context.Context, *domainaccount.Account) error { return nil }

func (r *skrAccountRepo) RegisterDevice(_ context.Context, accountDevice *domainaccount.AccountDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	acc, ok := r.byID[accountDevice.AccountID]
	if !ok {
		return domainaccount.ErrAccountNotFound
	}
	for i := range acc.Devices {
		if acc.Devices[i].DeviceID == accountDevice.DeviceID {
			acc.Devices[i] = *accountDevice
			return nil
		}
	}
	acc.Devices = append(acc.Devices, *accountDevice)
	return nil
}

func (r *skrAccountRepo) UpdateDeviceStatus(_ context.Context, accountID sharedDomain.AccountID, deviceID sharedDomain.DeviceID, status domainaccount.DeviceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	acc, ok := r.byID[accountID]
	if !ok {
		return domainaccount.ErrAccountNotFound
	}
	for i := range acc.Devices {
		if acc.Devices[i].DeviceID == deviceID {
			acc.Devices[i].Status = status
			return nil
		}
	}
	return nil
}

func (r *skrAccountRepo) RecordLoginEvent(context.Context, *domainaccount.AccountLoginEvent) error { return nil }
func (r *skrAccountRepo) ReplaceDevices(_ context.Context, accountID sharedDomain.AccountID, devices []domainaccount.AccountDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	acc, ok := r.byID[accountID]
	if !ok {
		return domainaccount.ErrAccountNotFound
	}
	acc.Devices = append([]domainaccount.AccountDevice(nil), devices...)
	return nil
}

func (r *skrAccountRepo) setFailFindByID(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failFindByID = enabled
}

func (r *skrAccountRepo) seedDevice(accountID sharedDomain.AccountID, deviceID sharedDomain.DeviceID, status domainaccount.DeviceStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	acc, ok := r.byID[accountID]
	if !ok {
		acc = &domainaccount.Account{
			ID:        accountID,
			Status:    domainaccount.Active,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		r.byID[accountID] = acc
	}
	for i := range acc.Devices {
		if acc.Devices[i].DeviceID == deviceID {
			acc.Devices[i].Status = status
			return
		}
	}
	acc.Devices = append(acc.Devices, domainaccount.AccountDevice{
		AccountID:  accountID,
		DeviceID:   deviceID,
		Status:     status,
		LastSeenAt: time.Now(),
	})
}

// ─── user repo ────────────────────────────────────────────────────────────────

type skrUserRepo struct {
	mu   sync.Mutex
	byID map[sharedDomain.UserID]*domainuser.User
}

func (r *skrUserRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = map[sharedDomain.UserID]*domainuser.User{}
}

func (r *skrUserRepo) Create(_ context.Context, user *domainuser.User) (sharedDomain.UserID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[user.ID] = user
	return user.ID, nil
}

func (r *skrUserRepo) FindByID(_ context.Context, id sharedDomain.UserID) (*domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, domainuser.ErrUserNotFound
	}
	copied := *u
	return &copied, nil
}

func (r *skrUserRepo) FindByAccountID(_ context.Context, accountID sharedDomain.AccountID) (*[]domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domainuser.User, 0)
	for _, user := range r.byID {
		if user.AccountID == accountID {
			out = append(out, *user)
		}
	}
	return &out, nil
}

func (r *skrUserRepo) Update(context.Context, *domainuser.User) error { return nil }
func (r *skrUserRepo) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}
func (r *skrUserRepo) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}
func (r *skrUserRepo) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

// ─── participant repo ─────────────────────────────────────────────────────────

type skrParticipantRepo struct {
	mu       sync.Mutex
	byUserID map[sharedDomain.UserID]*participant.Participant
	byID     map[participant.ID]*participant.Participant
}

func (r *skrParticipantRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byUserID = map[sharedDomain.UserID]*participant.Participant{}
	r.byID = map[participant.ID]*participant.Participant{}
}

func (r *skrParticipantRepo) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[id]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *skrParticipantRepo) FindByUserID(_ context.Context, userID sharedDomain.UserID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUserID[userID]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *skrParticipantRepo) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *skrParticipantRepo) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (r *skrParticipantRepo) Create(_ context.Context, p *participant.Participant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[p.ID] = p
	if p.UserID != nil {
		r.byUserID[*p.UserID] = p
	}
	return nil
}

// ─── chat member repo ─────────────────────────────────────────────────────────

type skrChatMemberRepo struct {
	mu                   sync.Mutex
	byID                 map[chatmember.ID]*chatmember.ChatMember
	byRoomAndParticipant map[chatroom.ID]map[participant.ID]*chatmember.ChatMember
}

func (r *skrChatMemberRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = map[chatmember.ID]*chatmember.ChatMember{}
	r.byRoomAndParticipant = map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{}
}

func (r *skrChatMemberRepo) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	copied := *m
	return &copied, nil
}

func (r *skrChatMemberRepo) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, pID participant.ID) (*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rm, ok := r.byRoomAndParticipant[roomID]; ok {
		if m, ok := rm[pID]; ok {
			copied := *m
			return &copied, nil
		}
	}
	return nil, chatmember.ErrNotFound
}

func (r *skrChatMemberRepo) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatmember.ChatMember, 0)
	for _, m := range r.byID {
		if m.RoomID == roomID {
			copied := *m
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *skrChatMemberRepo) FindByParticipant(_ context.Context, pID participant.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatmember.ChatMember, 0)
	for _, m := range r.byID {
		if m.ParticipantID == pID {
			copied := *m
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *skrChatMemberRepo) Add(_ context.Context, m *chatmember.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[m.ID] = m
	return nil
}

func (r *skrChatMemberRepo) Update(_ context.Context, m *chatmember.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[m.ID] = m
	return nil
}

func (r *skrChatMemberRepo) SoftDelete(_ context.Context, id chatmember.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.byID[id]; ok {
		m.IsDeleted = true
	}
	return nil
}

func (r *skrChatMemberRepo) Remove(_ context.Context, roomID chatroom.ID, pID participant.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rm, ok := r.byRoomAndParticipant[roomID]; ok {
		delete(rm, pID)
	}
	return nil
}

// ─── sender key request repo ──────────────────────────────────────────────────

type skrSenderKeyRequestRepo struct {
	mu      sync.Mutex
	records map[string]*senderkeyrequest.SenderKeyRequest
}

func (r *skrSenderKeyRequestRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = map[string]*senderkeyrequest.SenderKeyRequest{}
}

func (r *skrSenderKeyRequestRepo) key(
	requesterID chatmember.ID,
	requesterDeviceID sharedDomain.DeviceID,
	providerID chatmember.ID,
	providerDeviceID sharedDomain.DeviceID,
) string {
	return fmt.Sprintf("%d_%s_%d_%s", requesterID, requesterDeviceID.String(), providerID, providerDeviceID.String())
}

func (r *skrSenderKeyRequestRepo) Upsert(_ context.Context, req *senderkeyrequest.SenderKeyRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(req.RequesterMemberID, req.RequesterDeviceID, req.ProviderMemberID, req.ProviderDeviceID)
	copied := *req
	r.records[k] = &copied
	return nil
}

func (r *skrSenderKeyRequestRepo) FindPendingByProvider(_ context.Context, providerID chatmember.ID, providerDeviceID sharedDomain.DeviceID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*senderkeyrequest.SenderKeyRequest, 0)
	for _, req := range r.records {
		if req.ProviderMemberID == providerID && req.ProviderDeviceID == providerDeviceID && req.FulfilledAt == nil {
			copied := *req
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *skrSenderKeyRequestRepo) MarkFulfilled(
	_ context.Context,
	requesterID chatmember.ID,
	requesterDeviceID sharedDomain.DeviceID,
	providerID chatmember.ID,
	providerDeviceID sharedDomain.DeviceID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req, ok := r.records[r.key(requesterID, requesterDeviceID, providerID, providerDeviceID)]; ok {
		now := time.Now()
		req.FulfilledAt = &now
	}
	return nil
}

func (r *skrSenderKeyRequestRepo) find(requesterID, providerID chatmember.ID) (*senderkeyrequest.SenderKeyRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, req := range r.records {
		if req.RequesterMemberID == requesterID && req.ProviderMemberID == providerID {
			copied := *req
			return &copied, true
		}
	}
	return nil, false
}

func (r *skrSenderKeyRequestRepo) countPending(requesterID, providerID chatmember.ID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, req := range r.records {
		if req.RequesterMemberID == requesterID && req.ProviderMemberID == providerID && req.FulfilledAt == nil {
			count++
		}
	}
	return count
}

func (r *skrSenderKeyRequestRepo) countFulfilled(requesterID, providerID chatmember.ID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, req := range r.records {
		if req.RequesterMemberID == requesterID && req.ProviderMemberID == providerID && req.FulfilledAt != nil {
			count++
		}
	}
	return count
}

// ─── member sender key repo ───────────────────────────────────────────────────

type skrMemberSenderKeyRepo struct {
	mu   sync.Mutex
	keys map[string]*membersenderkey.MemberSenderKey
}

type skrSenderKeyReceiptRepo struct {
	mu      sync.Mutex
	records map[string]*senderkeyreceipt.SenderKeyReceipt
}

func (r *skrMemberSenderKeyRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys = map[string]*membersenderkey.MemberSenderKey{}
}

func (r *skrMemberSenderKeyRepo) key(memberID chatmember.ID, senderDeviceID sharedDomain.DeviceID) string {
	return fmt.Sprintf("%d_%s", memberID, senderDeviceID.String())
}

func (r *skrMemberSenderKeyRepo) FindLatest(_ context.Context, memberID chatmember.ID, senderDeviceID sharedDomain.DeviceID) (*membersenderkey.MemberSenderKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[r.key(memberID, senderDeviceID)]
	if !ok {
		return nil, membersenderkey.ErrNotFound
	}
	copied := *k
	return &copied, nil
}

func (r *skrMemberSenderKeyRepo) FindLatestForMember(_ context.Context, memberID chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*membersenderkey.MemberSenderKey, 0)
	for _, k := range r.keys {
		if k.ChatMemberID != memberID {
			continue
		}
		copied := *k
		out = append(out, &copied)
	}
	if len(out) == 0 {
		return nil, membersenderkey.ErrNotFound
	}
	return out, nil
}

func (r *skrMemberSenderKeyRepo) FindAllByMembers(_ context.Context, ids []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*membersenderkey.MemberSenderKey, 0)
	for _, k := range r.keys {
		for _, id := range ids {
			if k.ChatMemberID == id {
				copied := *k
				out = append(out, &copied)
				break
			}
		}
	}
	return out, nil
}

func (r *skrMemberSenderKeyRepo) Add(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *sk
	r.keys[r.key(sk.ChatMemberID, sk.SenderDeviceID)] = &copied
	return nil
}

func (r *skrMemberSenderKeyRepo) UpsertLatest(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *sk
	r.keys[r.key(sk.ChatMemberID, sk.SenderDeviceID)] = &copied
	return nil
}

func (r *skrSenderKeyReceiptRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = map[string]*senderkeyreceipt.SenderKeyReceipt{}
}

func (r *skrSenderKeyReceiptRepo) key(
	senderMemberID chatmember.ID,
	senderDeviceID sharedDomain.DeviceID,
	receiverMemberID chatmember.ID,
	receiverDeviceID sharedDomain.DeviceID,
) string {
	return fmt.Sprintf("%d_%s_%d_%s", senderMemberID, senderDeviceID.String(), receiverMemberID, receiverDeviceID.String())
}

func (r *skrSenderKeyReceiptRepo) FindLatest(
	_ context.Context,
	senderMemberID chatmember.ID,
	senderDeviceID sharedDomain.DeviceID,
	receiverMemberID chatmember.ID,
	receiverDeviceID sharedDomain.DeviceID,
) (*senderkeyreceipt.SenderKeyReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	receipt, ok := r.records[r.key(senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID)]
	if !ok {
		return nil, senderkeyreceipt.ErrNotFound
	}
	copied := *receipt
	return &copied, nil
}

func (r *skrSenderKeyReceiptRepo) Upsert(_ context.Context, receipt *senderkeyreceipt.SenderKeyReceipt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *receipt
	if copied.UpdatedAt.IsZero() {
		copied.UpdatedAt = time.Now()
	}
	r.records[r.key(receipt.SenderMemberID, receipt.SenderDeviceID, receipt.ReceiverMemberID, receipt.ReceiverDeviceID)] = &copied
	return nil
}

// ─── sender key distribution repo ────────────────────────────────────────────

type skrSenderKeyDistributionRepo struct {
	mu     sync.Mutex
	byPair map[string]*senderkeydistribution.SenderKeyDistribution
	byID   map[senderkeydistribution.ID]*senderkeydistribution.SenderKeyDistribution
	nextID senderkeydistribution.ID
}

func (r *skrSenderKeyDistributionRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byPair = map[string]*senderkeydistribution.SenderKeyDistribution{}
	r.byID = map[senderkeydistribution.ID]*senderkeydistribution.SenderKeyDistribution{}
	r.nextID = 0
}

func (r *skrSenderKeyDistributionRepo) key(senderID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) string {
	return fmt.Sprintf("%d_%s_%d_%s", senderID, senderDeviceID.String(), receiverID, receiverDeviceID.String())
}

func (r *skrSenderKeyDistributionRepo) clone(dist *senderkeydistribution.SenderKeyDistribution) *senderkeydistribution.SenderKeyDistribution {
	if dist == nil {
		return nil
	}
	copied := *dist
	copied.DistributionMessage = append([]byte(nil), dist.DistributionMessage...)
	return &copied
}

func (r *skrSenderKeyDistributionRepo) seed(roomID int64, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	dist := &senderkeydistribution.SenderKeyDistribution{
		ID:               r.nextID,
		RoomID:           roomID,
		SenderMemberID:   senderMemberID,
		SenderDeviceID:   senderDeviceID,
		ReceiverMemberID: receiverMemberID,
		ReceiverDeviceID: receiverDeviceID,
		SenderKeyVersion: version,
		Status:           status,
		ChainID:          version,
		DistributedAt:    time.Now(),
	}
	r.byPair[r.key(senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID)] = dist
	r.byID[dist.ID] = dist
	return dist.ID
}

func (r *skrSenderKeyDistributionRepo) findLatest(senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, dist := range r.byPair {
		if dist.SenderMemberID == senderMemberID && dist.ReceiverMemberID == receiverMemberID {
			return r.clone(dist), true
		}
	}
	return nil, false
}

func (r *skrSenderKeyDistributionRepo) findByID(id senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	return r.clone(dist), true
}

func (r *skrSenderKeyDistributionRepo) UpsertBatch(_ context.Context, dists []*senderkeydistribution.SenderKeyDistribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, dist := range dists {
		k := r.key(dist.SenderMemberID, dist.SenderDeviceID, dist.ReceiverMemberID, dist.ReceiverDeviceID)
		existing, ok := r.byPair[k]
		if ok {
			existing.ChainID = dist.ChainID
			existing.SenderKeyVersion = dist.ChainID
			existing.Status = senderkeydistribution.StatusConsumed
			existing.SenderDeviceID = dist.SenderDeviceID
			now := time.Now()
			existing.ConsumedAt = &now
			continue
		}
		r.nextID++
		copied := *dist
		copied.ID = r.nextID
		copied.Status = senderkeydistribution.StatusConsumed
		copied.SenderKeyVersion = dist.ChainID
		now := time.Now()
		copied.ConsumedAt = &now
		r.byPair[k] = &copied
		r.byID[copied.ID] = &copied
	}
	return nil
}

func (r *skrSenderKeyDistributionRepo) FindPendingReceivers(_ context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, latestChainID int64) ([]chatmember.ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]chatmember.ID, 0)
	for _, dist := range r.byPair {
		if dist.SenderMemberID != senderMemberID || dist.SenderDeviceID != senderDeviceID {
			continue
		}
		if dist.Status != senderkeydistribution.StatusConsumed || dist.SenderKeyVersion < latestChainID {
			out = append(out, dist.ReceiverMemberID)
		}
	}
	return out, nil
}

func (r *skrSenderKeyDistributionRepo) UpsertAvailable(_ context.Context, dist *senderkeydistribution.SenderKeyDistribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(dist.SenderMemberID, dist.SenderDeviceID, dist.ReceiverMemberID, dist.ReceiverDeviceID)
	if existing, ok := r.byPair[k]; ok {
		existing.RoomID = dist.RoomID
		existing.SenderDeviceID = dist.SenderDeviceID
		existing.SenderKeyVersion = dist.SenderKeyVersion
		existing.ChainID = dist.ChainID
		existing.DistributionMessage = append([]byte(nil), dist.DistributionMessage...)
		existing.Status = senderkeydistribution.StatusAvailable
		existing.DistributedAt = time.Now()
		existing.ConsumedAt = nil
		existing.FailedAt = nil
		dist.ID = existing.ID
		return nil
	}
	r.nextID++
	copied := r.clone(dist)
	copied.ID = r.nextID
	copied.Status = senderkeydistribution.StatusAvailable
	copied.DistributedAt = time.Now()
	r.byPair[k] = copied
	r.byID[copied.ID] = copied
	dist.ID = copied.ID
	return nil
}

func (r *skrSenderKeyDistributionRepo) FindLatest(_ context.Context, senderMemberID chatmember.ID, senderDeviceID sharedDomain.DeviceID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) (*senderkeydistribution.SenderKeyDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if dist, ok := r.byPair[r.key(senderMemberID, senderDeviceID, receiverMemberID, receiverDeviceID)]; ok {
		return r.clone(dist), nil
	}
	for _, dist := range r.byPair {
		if dist.SenderMemberID == senderMemberID && dist.SenderDeviceID == senderDeviceID && dist.ReceiverMemberID == receiverMemberID {
			if dist.ReceiverDeviceID == receiverDeviceID {
				return r.clone(dist), nil
			}
		}
	}
	return nil, senderkeydistribution.ErrNotFound
}

func (r *skrSenderKeyDistributionRepo) FindAvailableByRoomAndReceiver(_ context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID, receiverDeviceID sharedDomain.DeviceID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*senderkeydistribution.SenderKeyDistribution, 0)
	for _, dist := range r.byPair {
		if dist.RoomID != int64(roomID) || dist.ReceiverMemberID != receiverMemberID || dist.Status != senderkeydistribution.StatusAvailable {
			continue
		}
		if receiverDeviceID != (sharedDomain.DeviceID{}) && dist.ReceiverDeviceID != (sharedDomain.DeviceID{}) && dist.ReceiverDeviceID != receiverDeviceID {
			continue
		}
		out = append(out, r.clone(dist))
	}
	return out, nil
}

func (r *skrSenderKeyDistributionRepo) FindByID(_ context.Context, id senderkeydistribution.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byID[id]
	if !ok {
		return nil, senderkeydistribution.ErrNotFound
	}
	return r.clone(dist), nil
}

func (r *skrSenderKeyDistributionRepo) MarkConsumed(_ context.Context, id senderkeydistribution.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byID[id]
	if !ok {
		return senderkeydistribution.ErrNotFound
	}
	now := time.Now()
	dist.Status = senderkeydistribution.StatusConsumed
	dist.ConsumedAt = &now
	dist.FailedAt = nil
	return nil
}

func (r *skrSenderKeyDistributionRepo) MarkFailed(_ context.Context, id senderkeydistribution.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byID[id]
	if !ok {
		return senderkeydistribution.ErrNotFound
	}
	now := time.Now()
	dist.Status = senderkeydistribution.StatusFailed
	dist.FailedAt = &now
	dist.ConsumedAt = nil
	return nil
}

// ─── friendship repo ──────────────────────────────────────────────────────────

type skrFriendshipRepo struct {
	mu   sync.Mutex
	rows []*domainfriendship.Friendship
}

type skrDeviceRepo struct {
	mu   sync.Mutex
	byID map[sharedDomain.DeviceID]*domaindevice.Device
}

func (r *skrDeviceRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = map[sharedDomain.DeviceID]*domaindevice.Device{}
}

func (r *skrDeviceRepo) seed(deviceData *domaindevice.Device) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *deviceData
	r.byID[deviceData.ID] = &copied
}

func (r *skrDeviceRepo) FindByID(_ context.Context, deviceID sharedDomain.DeviceID) (*domaindevice.Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deviceData, ok := r.byID[deviceID]
	if !ok {
		return nil, domaindevice.ErrDeviceNotFound
	}
	copied := *deviceData
	return &copied, nil
}

func (r *skrDeviceRepo) FindAllByAccountID(context.Context, sharedDomain.AccountID) ([]*domaindevice.Device, error) {
	return nil, nil
}

func (r *skrDeviceRepo) Create(context.Context, *domaindevice.Device) error { return nil }
func (r *skrDeviceRepo) Update(context.Context, *domaindevice.Device) error { return nil }
func (r *skrDeviceRepo) BindAccount(context.Context, sharedDomain.DeviceID, sharedDomain.AccountID) error {
	return nil
}
func (r *skrDeviceRepo) DeleteByAccount(context.Context, sharedDomain.DeviceID, sharedDomain.AccountID) error {
	return nil
}

type skrSelfSenderKeySyncRepo struct {
	mu              sync.Mutex
	byParticipantID map[participant.ID]*selfsenderkeysync.SelfSenderKeySync
	nextID          selfsenderkeysync.ID
}

type skrSelfSenderKeySyncDistributionRepo struct {
	mu     sync.Mutex
	rows   map[string]map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution
	nextID selfsenderkeysyncdistribution.ID
}

func (r *skrSelfSenderKeySyncRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byParticipantID = map[participant.ID]*selfsenderkeysync.SelfSenderKeySync{}
	r.nextID = 0
}

func (r *skrSelfSenderKeySyncRepo) seed(syncState *selfsenderkeysync.SelfSenderKeySync) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *syncState
	if copied.ID == 0 {
		r.nextID++
		copied.ID = r.nextID
	} else if copied.ID > r.nextID {
		r.nextID = copied.ID
	}
	if syncState.ProviderDeviceID != nil {
		providerDeviceID := *syncState.ProviderDeviceID
		copied.ProviderDeviceID = &providerDeviceID
	}
	r.byParticipantID[syncState.ParticipantID] = &copied
}

func (r *skrSelfSenderKeySyncRepo) FindByParticipantID(_ context.Context, participantID participant.ID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	syncState, ok := r.byParticipantID[participantID]
	if !ok {
		return nil, selfsenderkeysync.ErrNotFound
	}
	copied := *syncState
	if syncState.ProviderDeviceID != nil {
		providerDeviceID := *syncState.ProviderDeviceID
		copied.ProviderDeviceID = &providerDeviceID
	}
	if syncState.ProviderClaimedAt != nil {
		claimedAt := *syncState.ProviderClaimedAt
		copied.ProviderClaimedAt = &claimedAt
	}
	if syncState.UploadedAt != nil {
		uploadedAt := *syncState.UploadedAt
		copied.UploadedAt = &uploadedAt
	}
	if syncState.CompletedAt != nil {
		completedAt := *syncState.CompletedAt
		copied.CompletedAt = &completedAt
	}
	if syncState.FailedAt != nil {
		failedAt := *syncState.FailedAt
		copied.FailedAt = &failedAt
	}
	if syncState.LastError != nil {
		lastError := *syncState.LastError
		copied.LastError = &lastError
	}
	return &copied, nil
}

func (r *skrSelfSenderKeySyncRepo) UpsertPending(_ context.Context, participantID participant.ID, requesterDeviceID sharedDomain.DeviceID) (*selfsenderkeysync.SelfSenderKeySync, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	r.nextID++
	syncState := &selfsenderkeysync.SelfSenderKeySync{
		ID:                r.nextID,
		ParticipantID:     participantID,
		RequesterDeviceID: requesterDeviceID,
		Status:            selfsenderkeysync.StatusPendingProvider,
		RequestedAt:       now,
		UpdatedAt:         now,
	}
	r.byParticipantID[participantID] = syncState
	copied := *syncState
	return &copied, nil
}

func (r *skrSelfSenderKeySyncRepo) ClaimProvider(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID sharedDomain.DeviceID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	syncState, ok := r.byParticipantID[participantID]
	if !ok || syncState.ID != id || syncState.RequesterDeviceID != requesterDeviceID || syncState.Status != selfsenderkeysync.StatusPendingProvider {
		return false, nil
	}
	now := time.Now()
	syncState.ProviderDeviceID = &providerDeviceID
	syncState.ProviderClaimedAt = &now
	syncState.Status = selfsenderkeysync.StatusSyncing
	syncState.UpdatedAt = now
	return true, nil
}

func (r *skrSelfSenderKeySyncRepo) MarkUploaded(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID sharedDomain.DeviceID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	syncState, ok := r.byParticipantID[participantID]
	if !ok || syncState.ID != id || syncState.RequesterDeviceID != requesterDeviceID || syncState.ProviderDeviceID == nil || *syncState.ProviderDeviceID != providerDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	syncState.Status = selfsenderkeysync.StatusUploaded
	syncState.UploadedAt = &now
	syncState.UpdatedAt = now
	return nil
}

func (r *skrSelfSenderKeySyncRepo) MarkCompleted(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID sharedDomain.DeviceID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	syncState, ok := r.byParticipantID[participantID]
	if !ok || syncState.ID != id || syncState.RequesterDeviceID != requesterDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	syncState.Status = selfsenderkeysync.StatusCompleted
	syncState.CompletedAt = &now
	syncState.UpdatedAt = now
	return nil
}

func (r *skrSelfSenderKeySyncRepo) MarkFailed(_ context.Context, id selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID, providerDeviceID sharedDomain.DeviceID, lastError string, retryable bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	syncState, ok := r.byParticipantID[participantID]
	if !ok || syncState.ID != id || syncState.RequesterDeviceID != requesterDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	if syncState.ProviderDeviceID != nil && providerDeviceID != (sharedDomain.DeviceID{}) && *syncState.ProviderDeviceID != providerDeviceID {
		return selfsenderkeysync.ErrNotFound
	}
	now := time.Now()
	if retryable {
		syncState.Status = selfsenderkeysync.StatusPendingProvider
		syncState.ProviderDeviceID = nil
	} else {
		syncState.Status = selfsenderkeysync.StatusFailed
	}
	if lastError != "" {
		errCopy := lastError
		syncState.LastError = &errCopy
	} else {
		syncState.LastError = nil
	}
	syncState.FailedAt = &now
	syncState.UpdatedAt = now
	return nil
}

func (r *skrSelfSenderKeySyncDistributionRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = map[string]map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{}
	r.nextID = 0
}

func (r *skrSelfSenderKeySyncDistributionRepo) key(selfSyncID selfsenderkeysync.ID, participantID participant.ID, requesterDeviceID sharedDomain.DeviceID) string {
	return fmt.Sprintf("%d_%d_%s", selfSyncID, participantID, requesterDeviceID.String())
}

func (r *skrSelfSenderKeySyncDistributionRepo) clone(row *selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution) *selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution {
	if row == nil {
		return nil
	}
	copied := *row
	copied.DistributionMessage = append([]byte(nil), row.DistributionMessage...)
	return &copied
}

func (r *skrSelfSenderKeySyncDistributionRepo) ReplaceForSync(
	_ context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID sharedDomain.DeviceID,
	providerDeviceID sharedDomain.DeviceID,
	rows []*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.key(selfSyncID, participantID, requesterDeviceID)
	r.rows[key] = map[selfsenderkeysyncdistribution.ID]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution{}
	for _, row := range rows {
		r.nextID++
		copied := r.clone(row)
		copied.ID = r.nextID
		copied.SelfSenderKeySyncID = selfSyncID
		copied.ParticipantID = participantID
		copied.RequesterDeviceID = requesterDeviceID
		copied.ProviderDeviceID = providerDeviceID
		if copied.Status == "" {
			copied.Status = selfsenderkeysyncdistribution.StatusAvailable
		}
		if copied.CreatedAt.IsZero() {
			copied.CreatedAt = time.Now()
		}
		r.rows[key][copied.ID] = copied
	}
	return nil
}

func (r *skrSelfSenderKeySyncDistributionRepo) FindPendingByRequester(
	_ context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID sharedDomain.DeviceID,
) ([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*selfsenderkeysyncdistribution.SelfSenderKeySyncDistribution, 0)
	for _, row := range r.rows[r.key(selfSyncID, participantID, requesterDeviceID)] {
		if row.Status != selfsenderkeysyncdistribution.StatusAvailable {
			continue
		}
		out = append(out, r.clone(row))
	}
	return out, nil
}

func (r *skrSelfSenderKeySyncDistributionRepo) MarkConsumed(
	_ context.Context,
	selfSyncID selfsenderkeysync.ID,
	id selfsenderkeysyncdistribution.ID,
	participantID participant.ID,
	requesterDeviceID sharedDomain.DeviceID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[r.key(selfSyncID, participantID, requesterDeviceID)][id]; ok {
		now := time.Now()
		row.Status = selfsenderkeysyncdistribution.StatusConsumed
		row.ConsumedAt = &now
		row.FailedAt = nil
	}
	return nil
}

func (r *skrSelfSenderKeySyncDistributionRepo) MarkFailed(
	_ context.Context,
	selfSyncID selfsenderkeysync.ID,
	id selfsenderkeysyncdistribution.ID,
	participantID participant.ID,
	requesterDeviceID sharedDomain.DeviceID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[r.key(selfSyncID, participantID, requesterDeviceID)][id]; ok {
		now := time.Now()
		row.Status = selfsenderkeysyncdistribution.StatusFailed
		row.FailedAt = &now
		row.ConsumedAt = nil
	}
	return nil
}

func (r *skrSelfSenderKeySyncDistributionRepo) HasNonConsumed(
	_ context.Context,
	selfSyncID selfsenderkeysync.ID,
	participantID participant.ID,
	requesterDeviceID sharedDomain.DeviceID,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, row := range r.rows[r.key(selfSyncID, participantID, requesterDeviceID)] {
		if row.Status != selfsenderkeysyncdistribution.StatusConsumed {
			return true, nil
		}
	}
	return false, nil
}

func (r *skrFriendshipRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = nil
}

func (r *skrFriendshipRepo) seedBlocked(userID, friendID sharedDomain.UserID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, &domainfriendship.Friendship{
		UserID:   userID,
		FriendID: friendID,
		Status:   domainfriendship.StatusBlocked,
	})
}

func (r *skrFriendshipRepo) FindBetweenUsers(_ context.Context, userID, friendID sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domainfriendship.Friendship, 0)
	for _, row := range r.rows {
		if (row.UserID == userID && row.FriendID == friendID) || (row.UserID == friendID && row.FriendID == userID) {
			copied := *row
			out = append(out, &copied)
		}
	}
	if len(out) == 0 {
		return nil, domainfriendship.ErrFriendshipNotFound
	}
	return out, nil
}

func (r *skrFriendshipRepo) FindByUserID(context.Context, sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, nil
}

func (r *skrFriendshipRepo) FindAllByUserID(context.Context, sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, nil
}

func (r *skrFriendshipRepo) FindPendingByUserID(context.Context, sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, nil
}

func (r *skrFriendshipRepo) Create(context.Context, sharedDomain.UserID, sharedDomain.UserID) error {
	return nil
}

func (r *skrFriendshipRepo) CreateBlocked(context.Context, sharedDomain.UserID, sharedDomain.UserID) error {
	return nil
}

func (r *skrFriendshipRepo) FindByID(context.Context, int64) (*domainfriendship.Friendship, error) {
	return nil, domainfriendship.ErrFriendshipNotFound
}

func (r *skrFriendshipRepo) FindByUserIDAndFriendID(context.Context, sharedDomain.UserID, sharedDomain.UserID) (*domainfriendship.Friendship, error) {
	return nil, domainfriendship.ErrFriendshipNotFound
}

func (r *skrFriendshipRepo) FindPendingByFriendID(context.Context, sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, nil
}

func (r *skrFriendshipRepo) UpdateStatus(context.Context, int64, domainfriendship.Status) error {
	return nil
}

func (r *skrFriendshipRepo) Delete(context.Context, int64) error {
	return nil
}

var (
	_ domainaccount.Repository         = (*skrAccountRepo)(nil)
	_ domainuser.Repository            = (*skrUserRepo)(nil)
	_ participant.Repository           = (*skrParticipantRepo)(nil)
	_ chatmember.Repository            = (*skrChatMemberRepo)(nil)
	_ senderkeyrequest.Repository      = (*skrSenderKeyRequestRepo)(nil)
	_ membersenderkey.Repository       = (*skrMemberSenderKeyRepo)(nil)
	_ senderkeyreceipt.Repository      = (*skrSenderKeyReceiptRepo)(nil)
	_ senderkeydistribution.Repository = (*skrSenderKeyDistributionRepo)(nil)
	_ selfsenderkeysync.Repository     = (*skrSelfSenderKeySyncRepo)(nil)
	_ selfsenderkeysyncdistribution.Repository = (*skrSelfSenderKeySyncDistributionRepo)(nil)
	_ domainfriendship.Repository      = (*skrFriendshipRepo)(nil)
)
