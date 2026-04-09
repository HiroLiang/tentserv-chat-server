package e2ee

import (
	"context"
	"fmt"
	"sync"
	"time"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	e2eeUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/usecase"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	domainfriendship "github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
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
	participantRepo  *skrParticipantRepo
	chatMemberRepo   *skrChatMemberRepo
	skrRepo          *skrSenderKeyRequestRepo
	mskRepo          *skrMemberSenderKeyRepo
	distributionRepo *skrSenderKeyDistributionRepo
	friendshipRepo   *skrFriendshipRepo
	broadcaster      *recordingBroadcaster
}

func NewSKRDeps() *SKRDeps {
	return &SKRDeps{
		participantRepo:  &skrParticipantRepo{byUserID: map[sharedDomain.UserID]*participant.Participant{}, byID: map[participant.ID]*participant.Participant{}},
		chatMemberRepo:   &skrChatMemberRepo{byID: map[chatmember.ID]*chatmember.ChatMember{}, byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{}},
		skrRepo:          &skrSenderKeyRequestRepo{records: map[string]*senderkeyrequest.SenderKeyRequest{}},
		mskRepo:          &skrMemberSenderKeyRepo{keys: map[chatmember.ID]*membersenderkey.MemberSenderKey{}},
		distributionRepo: &skrSenderKeyDistributionRepo{byPair: map[string]*senderkeydistribution.SenderKeyDistribution{}, byID: map[senderkeydistribution.ID]*senderkeydistribution.SenderKeyDistribution{}},
		friendshipRepo:   &skrFriendshipRepo{},
		broadcaster:      &recordingBroadcaster{},
	}
}

func (d *SKRDeps) Reset() {
	d.participantRepo.reset()
	d.chatMemberRepo.reset()
	d.skrRepo.reset()
	d.mskRepo.reset()
	d.distributionRepo.reset()
	d.friendshipRepo.reset()
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
		d.friendshipRepo,
		&noBroadcaster{},
	)
}

func (d *SKRDeps) RegisterUploadSenderKeyUseCase() *e2eeUseCase.UploadSenderKeyUseCase {
	return e2eeUseCase.NewUploadSenderKeyUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.mskRepo,
		d.distributionRepo,
		d.skrRepo,
		d.friendshipRepo,
		&noBroadcaster{},
	)
}

func (d *SKRDeps) RegisterGetSenderKeyDistributionStatusUseCase() *e2eeUseCase.GetSenderKeyDistributionStatusUseCase {
	return e2eeUseCase.NewGetSenderKeyDistributionStatusUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.mskRepo,
		d.distributionRepo,
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
		d.skrRepo,
		d.broadcaster,
	)
}

// SeedParticipant creates a participant for a user without adding room membership.
func (d *SKRDeps) SeedParticipant(userID sharedDomain.UserID, participantID participant.ID) {
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
	d.mskRepo.keys[memberID] = &membersenderkey.MemberSenderKey{
		ChatMemberID:     memberID,
		SenderKeyVersion: version,
		ChainID:          membersenderkey.ChainID(version),
	}
}

func (d *SKRDeps) SeedDistribution(roomID int64, senderMemberID, receiverMemberID chatmember.ID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	return d.distributionRepo.seed(roomID, senderMemberID, receiverMemberID, version, status)
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

func (r *skrSenderKeyRequestRepo) key(requesterID, providerID chatmember.ID) string {
	return fmt.Sprintf("%d_%d", requesterID, providerID)
}

func (r *skrSenderKeyRequestRepo) Upsert(_ context.Context, req *senderkeyrequest.SenderKeyRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(req.RequesterMemberID, req.ProviderMemberID)
	copied := *req
	r.records[k] = &copied
	return nil
}

func (r *skrSenderKeyRequestRepo) FindPendingByProvider(_ context.Context, providerID chatmember.ID) ([]*senderkeyrequest.SenderKeyRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*senderkeyrequest.SenderKeyRequest, 0)
	for _, req := range r.records {
		if req.ProviderMemberID == providerID && req.FulfilledAt == nil {
			copied := *req
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *skrSenderKeyRequestRepo) MarkFulfilled(_ context.Context, requesterID, providerID chatmember.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(requesterID, providerID)
	if req, ok := r.records[k]; ok {
		now := time.Now()
		req.FulfilledAt = &now
	}
	return nil
}

func (r *skrSenderKeyRequestRepo) find(requesterID, providerID chatmember.ID) (*senderkeyrequest.SenderKeyRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(requesterID, providerID)
	req, ok := r.records[k]
	if !ok {
		return nil, false
	}
	copied := *req
	return &copied, true
}

func (r *skrSenderKeyRequestRepo) countPending(requesterID, providerID chatmember.ID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.records[r.key(requesterID, providerID)]
	if !ok || req.FulfilledAt != nil {
		return 0
	}
	return 1
}

func (r *skrSenderKeyRequestRepo) countFulfilled(requesterID, providerID chatmember.ID) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, ok := r.records[r.key(requesterID, providerID)]
	if !ok || req.FulfilledAt == nil {
		return 0
	}
	return 1
}

// ─── member sender key repo ───────────────────────────────────────────────────

type skrMemberSenderKeyRepo struct {
	mu   sync.Mutex
	keys map[chatmember.ID]*membersenderkey.MemberSenderKey
}

func (r *skrMemberSenderKeyRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys = map[chatmember.ID]*membersenderkey.MemberSenderKey{}
}

func (r *skrMemberSenderKeyRepo) FindLatest(_ context.Context, memberID chatmember.ID) (*membersenderkey.MemberSenderKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[memberID]
	if !ok {
		return nil, membersenderkey.ErrNotFound
	}
	copied := *k
	return &copied, nil
}

func (r *skrMemberSenderKeyRepo) FindAllByMembers(_ context.Context, ids []chatmember.ID) ([]*membersenderkey.MemberSenderKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*membersenderkey.MemberSenderKey, 0)
	for _, id := range ids {
		if k, ok := r.keys[id]; ok {
			copied := *k
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *skrMemberSenderKeyRepo) Add(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[sk.ChatMemberID] = sk
	return nil
}

func (r *skrMemberSenderKeyRepo) UpsertLatest(_ context.Context, sk *membersenderkey.MemberSenderKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := *sk
	r.keys[sk.ChatMemberID] = &copied
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

func (r *skrSenderKeyDistributionRepo) key(senderID, receiverID chatmember.ID) string {
	return fmt.Sprintf("%d_%d", senderID, receiverID)
}

func (r *skrSenderKeyDistributionRepo) clone(dist *senderkeydistribution.SenderKeyDistribution) *senderkeydistribution.SenderKeyDistribution {
	if dist == nil {
		return nil
	}
	copied := *dist
	copied.DistributionMessage = append([]byte(nil), dist.DistributionMessage...)
	return &copied
}

func (r *skrSenderKeyDistributionRepo) seed(roomID int64, senderMemberID, receiverMemberID chatmember.ID, version int64, status senderkeydistribution.Status) senderkeydistribution.ID {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	dist := &senderkeydistribution.SenderKeyDistribution{
		ID:               r.nextID,
		RoomID:           roomID,
		SenderMemberID:   senderMemberID,
		ReceiverMemberID: receiverMemberID,
		SenderKeyVersion: version,
		Status:           status,
		ChainID:          int(version),
		DistributedAt:    time.Now(),
	}
	r.byPair[r.key(senderMemberID, receiverMemberID)] = dist
	r.byID[dist.ID] = dist
	return dist.ID
}

func (r *skrSenderKeyDistributionRepo) findLatest(senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byPair[r.key(senderMemberID, receiverMemberID)]
	if !ok {
		return nil, false
	}
	return r.clone(dist), true
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
		k := r.key(dist.SenderMemberID, dist.ReceiverMemberID)
		existing, ok := r.byPair[k]
		if ok {
			existing.ChainID = dist.ChainID
			existing.SenderKeyVersion = int64(dist.ChainID)
			existing.Status = senderkeydistribution.StatusConsumed
			now := time.Now()
			existing.ConsumedAt = &now
			continue
		}
		r.nextID++
		copied := *dist
		copied.ID = r.nextID
		copied.Status = senderkeydistribution.StatusConsumed
		copied.SenderKeyVersion = int64(dist.ChainID)
		now := time.Now()
		copied.ConsumedAt = &now
		r.byPair[k] = &copied
		r.byID[copied.ID] = &copied
	}
	return nil
}

func (r *skrSenderKeyDistributionRepo) FindPendingReceivers(_ context.Context, senderMemberID chatmember.ID, latestChainID int) ([]chatmember.ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]chatmember.ID, 0)
	for _, dist := range r.byPair {
		if dist.SenderMemberID != senderMemberID {
			continue
		}
		if dist.Status != senderkeydistribution.StatusConsumed || dist.SenderKeyVersion < int64(latestChainID) {
			out = append(out, dist.ReceiverMemberID)
		}
	}
	return out, nil
}

func (r *skrSenderKeyDistributionRepo) UpsertAvailable(_ context.Context, dist *senderkeydistribution.SenderKeyDistribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(dist.SenderMemberID, dist.ReceiverMemberID)
	if existing, ok := r.byPair[k]; ok {
		existing.RoomID = dist.RoomID
		existing.SenderKeyVersion = dist.SenderKeyVersion
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

func (r *skrSenderKeyDistributionRepo) FindLatest(_ context.Context, senderMemberID, receiverMemberID chatmember.ID) (*senderkeydistribution.SenderKeyDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	dist, ok := r.byPair[r.key(senderMemberID, receiverMemberID)]
	if !ok {
		return nil, senderkeydistribution.ErrNotFound
	}
	return r.clone(dist), nil
}

func (r *skrSenderKeyDistributionRepo) FindAvailableByRoomAndReceiver(_ context.Context, roomID chatroom.ID, receiverMemberID chatmember.ID) ([]*senderkeydistribution.SenderKeyDistribution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*senderkeydistribution.SenderKeyDistribution, 0)
	for _, dist := range r.byPair {
		if dist.RoomID != int64(roomID) || dist.ReceiverMemberID != receiverMemberID || dist.Status != senderkeydistribution.StatusAvailable {
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
	_ participant.Repository           = (*skrParticipantRepo)(nil)
	_ chatmember.Repository            = (*skrChatMemberRepo)(nil)
	_ senderkeyrequest.Repository      = (*skrSenderKeyRequestRepo)(nil)
	_ membersenderkey.Repository       = (*skrMemberSenderKeyRepo)(nil)
	_ senderkeydistribution.Repository = (*skrSenderKeyDistributionRepo)(nil)
	_ domainfriendship.Repository      = (*skrFriendshipRepo)(nil)
)
