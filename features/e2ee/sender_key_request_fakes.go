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
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	sharedDomain "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

// noBroadcaster is a no-op broadcaster for BDD tests.
type noBroadcaster struct{}

func (n *noBroadcaster) SendToUser(string, []byte) {}

var _ e2eePort.Broadcaster = (*noBroadcaster)(nil)

// SKRDeps holds the in-memory repositories needed for sender key request BDD scenarios.
type SKRDeps struct {
	mu              sync.Mutex
	participantRepo *skrParticipantRepo
	chatMemberRepo  *skrChatMemberRepo
	skrRepo         *skrSenderKeyRequestRepo
	mskRepo         *skrMemberSenderKeyRepo
	friendshipRepo  *skrFriendshipRepo
}

func NewSKRDeps() *SKRDeps {
	return &SKRDeps{
		participantRepo: &skrParticipantRepo{byUserID: map[sharedDomain.UserID]*participant.Participant{}, byID: map[participant.ID]*participant.Participant{}},
		chatMemberRepo:  &skrChatMemberRepo{byID: map[chatmember.ID]*chatmember.ChatMember{}, byRoomAndParticipant: map[chatroom.ID]map[participant.ID]*chatmember.ChatMember{}},
		skrRepo:         &skrSenderKeyRequestRepo{records: map[string]*senderkeyrequest.SenderKeyRequest{}},
		mskRepo:         &skrMemberSenderKeyRepo{keys: map[chatmember.ID]*membersenderkey.MemberSenderKey{}},
		friendshipRepo:  &skrFriendshipRepo{},
	}
}

func (d *SKRDeps) Reset() {
	d.participantRepo.reset()
	d.chatMemberRepo.reset()
	d.skrRepo.reset()
	d.mskRepo.reset()
}

func (d *SKRDeps) RegisterCreateSenderKeyRequestUseCase() *e2eeUseCase.CreateSenderKeyRequestUseCase {
	return e2eeUseCase.NewCreateSenderKeyRequestUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.skrRepo,
		d.mskRepo,
		d.friendshipRepo,
		&noBroadcaster{},
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
	d.mskRepo.keys[memberID] = &membersenderkey.MemberSenderKey{ChatMemberID: memberID}
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

// ─── friendship repo ──────────────────────────────────────────────────────────

type skrFriendshipRepo struct{}

func (r *skrFriendshipRepo) FindBetweenUsers(context.Context, sharedDomain.UserID, sharedDomain.UserID) ([]*domainfriendship.Friendship, error) {
	return nil, domainfriendship.ErrFriendshipNotFound
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
	_ participant.Repository      = (*skrParticipantRepo)(nil)
	_ chatmember.Repository       = (*skrChatMemberRepo)(nil)
	_ senderkeyrequest.Repository = (*skrSenderKeyRequestRepo)(nil)
	_ membersenderkey.Repository  = (*skrMemberSenderKeyRepo)(nil)
	_ domainfriendship.Repository = (*skrFriendshipRepo)(nil)
)
