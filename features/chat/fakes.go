package chat

import (
	"context"
	"io"
	"sync"
	"time"

	chatusecase "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	appPort "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/role"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type Deps struct {
	participantRepo *bddParticipantRepo
	chatMemberRepo  *bddChatMemberRepo
	chatRoomRepo    *bddChatRoomRepo
	chatMessageRepo *bddChatMessageRepo
	userRepo        *bddUserRepo
	agentRepo       *bddAgentRepo
}

func NewDeps() *Deps {
	deps := &Deps{
		participantRepo: newBDDParticipantRepo(),
		chatMemberRepo:  newBDDChatMemberRepo(),
		chatRoomRepo:    newBDDChatRoomRepo(),
		chatMessageRepo: newBDDChatMessageRepo(),
		userRepo:        newBDDUserRepo(),
		agentRepo:       &bddAgentRepo{},
	}
	deps.Reset()
	return deps
}

func (d *Deps) Reset() {
	d.participantRepo.reset()
	d.chatMemberRepo.reset()
	d.chatRoomRepo.reset()
	d.chatMessageRepo.reset()
	d.userRepo.reset()
}

func (d *Deps) RegisterGetUserChatRoomsUseCase() *chatusecase.GetUserChatRoomsUseCase {
	return chatusecase.NewGetUserChatRoomsUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.chatRoomRepo,
		d.chatMessageRepo,
		d.userRepo,
		d.agentRepo,
	)
}

func (d *Deps) RegisterGetChatRoomDetailUseCase() *chatusecase.GetChatRoomDetailUseCase {
	return chatusecase.NewGetChatRoomDetailUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.chatRoomRepo,
		d.chatMessageRepo,
		d.userRepo,
		d.agentRepo,
	)
}

func (d *Deps) RegisterGetChatRoomMessagesUseCase() *chatusecase.GetChatRoomMessagesUseCase {
	return chatusecase.NewGetChatRoomMessagesUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.chatRoomRepo,
		d.chatMessageRepo,
	)
}

func (d *Deps) RegisterUpdateMemberStatusUseCase() *chatusecase.UpdateMemberStatusUseCase {
	return chatusecase.NewUpdateMemberStatusUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.chatRoomRepo,
	)
}

func (d *Deps) RegisterSendMessageUseCase() *chatusecase.SendMessageUseCase {
	return chatusecase.NewSendMessageUseCase(
		d.participantRepo,
		d.chatMemberRepo,
		d.chatRoomRepo,
		d.chatMessageRepo,
		bddBroadcaster{},
	)
}

func (d *Deps) FileStorage() appPort.FileStorage {
	return bddFileStorage{}
}

func (d *Deps) SeedUser(userID shared.UserID, name, avatar string) {
	d.userRepo.seed(userID, name, avatar)
	d.participantRepo.seedForUser(userID)
}

func (d *Deps) CreateDirectRoomBetweenUsers(userID1, userID2 shared.UserID) chatroom.ID {
	p1 := d.participantRepo.findByUserID(userID1)
	p2 := d.participantRepo.findByUserID(userID2)
	if p1 == nil || p2 == nil {
		return 0
	}

	room := &chatroom.ChatRoom{
		Name:       "direct",
		Type:       chatroom.Direct,
		MaxMembers: 2,
		CreatedAt:  time.Now(),
	}
	_ = d.chatRoomRepo.Create(context.Background(), room)
	_ = d.chatMemberRepo.Add(context.Background(), &chatmember.ChatMember{
		RoomID:        room.ID,
		ParticipantID: p1.ID,
		Role:          chatmember.Owner,
		JoinedAt:      time.Now(),
	})
	_ = d.chatMemberRepo.Add(context.Background(), &chatmember.ChatMember{
		RoomID:        room.ID,
		ParticipantID: p2.ID,
		Role:          chatmember.Owner,
		JoinedAt:      time.Now(),
	})
	return room.ID
}

func (d *Deps) MarkRoomDeleted(roomID chatroom.ID) {
	_ = d.chatMemberRepo.SoftDeleteByRoom(context.Background(), roomID)
	_ = d.chatRoomRepo.SoftDelete(context.Background(), roomID)
}

func (d *Deps) SeedLatestMessage(roomID chatroom.ID, senderUserID shared.UserID, content string) chatmember.ID {
	p := d.participantRepo.findByUserID(senderUserID)
	if p == nil {
		return 0
	}
	member := d.chatMemberRepo.findByRoomAndParticipant(roomID, p.ID)
	if member == nil {
		return 0
	}
	_ = d.chatMessageRepo.Create(context.Background(), &chatmessage.ChatMessage{
		RoomID:    roomID,
		SenderID:  member.ID,
		Content:   content,
		Type:      chatmessage.Text,
		CreatedAt: time.Now(),
	})
	return member.ID
}

func (d *Deps) MemberIDForUser(roomID chatroom.ID, userID shared.UserID) chatmember.ID {
	p := d.participantRepo.findByUserID(userID)
	if p == nil {
		return 0
	}
	member := d.chatMemberRepo.findByRoomAndParticipant(roomID, p.ID)
	if member == nil {
		return 0
	}
	return member.ID
}

type bddUserRepo struct {
	mu        sync.Mutex
	usersByID map[shared.UserID]*domainuser.User
}

func newBDDUserRepo() *bddUserRepo {
	return &bddUserRepo{}
}

func (r *bddUserRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usersByID = map[shared.UserID]*domainuser.User{}
}

func (r *bddUserRepo) seed(userID shared.UserID, name, avatar string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usersByID[userID] = &domainuser.User{
		ID:        userID,
		Name:      name,
		Avatar:    avatar,
		RoleCodes: []role.Code{role.User},
	}
}

func (r *bddUserRepo) Create(context.Context, *domainuser.User) (shared.UserID, error) {
	return 0, nil
}

func (r *bddUserRepo) FindByID(_ context.Context, id shared.UserID) (*domainuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.usersByID[id]
	if !ok {
		return nil, domainuser.ErrUserNotFound
	}
	copied := *user
	copied.RoleCodes = append([]role.Code(nil), user.RoleCodes...)
	return &copied, nil
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

type bddParticipantRepo struct {
	mu     sync.Mutex
	nextID participant.ID
	byID   map[participant.ID]*participant.Participant
	byUser map[shared.UserID]*participant.Participant
}

func newBDDParticipantRepo() *bddParticipantRepo {
	return &bddParticipantRepo{}
}

func (r *bddParticipantRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[participant.ID]*participant.Participant{}
	r.byUser = map[shared.UserID]*participant.Participant{}
}

func (r *bddParticipantRepo) seedForUser(userID shared.UserID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byUser[userID]; ok {
		return
	}
	p := &participant.Participant{
		ID:        r.nextID,
		Type:      participant.UserType,
		UserID:    &userID,
		CreatedAt: time.Now(),
	}
	r.nextID++
	r.byID[p.ID] = p
	r.byUser[userID] = p
}

func (r *bddParticipantRepo) findByUserID(userID shared.UserID) *participant.Participant {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUser[userID]
	if !ok {
		return nil
	}
	copied := *p
	return &copied
}

func (r *bddParticipantRepo) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[id]
	if !ok {
		return nil, participant.ErrNotFound
	}
	copied := *p
	return &copied, nil
}

func (r *bddParticipantRepo) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byUser[userID]
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

func (r *bddParticipantRepo) Create(context.Context, *participant.Participant) error {
	return nil
}

type bddChatRoomRepo struct {
	mu     sync.Mutex
	nextID chatroom.ID
	byID   map[chatroom.ID]*chatroom.ChatRoom
}

func newBDDChatRoomRepo() *bddChatRoomRepo {
	return &bddChatRoomRepo{}
}

func (r *bddChatRoomRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[chatroom.ID]*chatroom.ChatRoom{}
}

func (r *bddChatRoomRepo) FindByID(_ context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	room, ok := r.byID[id]
	if !ok || room.IsDeleted {
		return nil, chatroom.ErrNotFound
	}
	copied := *room
	return &copied, nil
}

func (r *bddChatRoomRepo) Create(_ context.Context, room *chatroom.ChatRoom) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if room.ID == 0 {
		room.ID = r.nextID
		r.nextID++
	}
	copied := *room
	r.byID[room.ID] = &copied
	return nil
}

func (r *bddChatRoomRepo) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}

func (r *bddChatRoomRepo) Update(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (r *bddChatRoomRepo) SoftDelete(_ context.Context, id chatroom.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	room, ok := r.byID[id]
	if !ok {
		return chatroom.ErrNotFound
	}
	room.IsDeleted = true
	room.UpdatedAt = time.Now()
	return nil
}

type bddChatMemberRepo struct {
	mu     sync.Mutex
	nextID chatmember.ID
	byID   map[chatmember.ID]*chatmember.ChatMember
}

func newBDDChatMemberRepo() *bddChatMemberRepo {
	return &bddChatMemberRepo{}
}

func (r *bddChatMemberRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byID = map[chatmember.ID]*chatmember.ChatMember{}
}

func (r *bddChatMemberRepo) Add(_ context.Context, m *chatmember.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.ID == 0 {
		m.ID = r.nextID
		r.nextID++
	}
	copied := *m
	if copied.JoinedAt.IsZero() {
		copied.JoinedAt = time.Now()
	}
	r.byID[copied.ID] = &copied
	m.ID = copied.ID
	return nil
}

func (r *bddChatMemberRepo) findByRoomAndParticipant(roomID chatroom.ID, participantID participant.ID) *chatmember.ChatMember {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, member := range r.byID {
		if member.RoomID == roomID && member.ParticipantID == participantID && !member.IsDeleted {
			copied := *member
			return &copied
		}
	}
	return nil
}

func (r *bddChatMemberRepo) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	member, ok := r.byID[id]
	if !ok {
		return nil, chatmember.ErrNotFound
	}
	copied := *member
	return &copied, nil
}

func (r *bddChatMemberRepo) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	member := r.findByRoomAndParticipant(roomID, participantID)
	if member == nil {
		return nil, chatmember.ErrNotFound
	}
	return member, nil
}

func (r *bddChatMemberRepo) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatmember.ChatMember, 0)
	for _, member := range r.byID {
		if member.RoomID == roomID && !member.IsDeleted {
			copied := *member
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *bddChatMemberRepo) FindByParticipant(_ context.Context, participantID participant.ID) ([]*chatmember.ChatMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*chatmember.ChatMember, 0)
	for _, member := range r.byID {
		if member.ParticipantID == participantID && !member.IsDeleted {
			copied := *member
			out = append(out, &copied)
		}
	}
	return out, nil
}

func (r *bddChatMemberRepo) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (r *bddChatMemberRepo) SoftDelete(_ context.Context, id chatmember.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	member, ok := r.byID[id]
	if !ok {
		return chatmember.ErrNotFound
	}
	now := time.Now()
	member.IsDeleted = true
	member.DeletedAt = &now
	member.UpdatedAt = now
	return nil
}

func (r *bddChatMemberRepo) SoftDeleteByRoom(_ context.Context, roomID chatroom.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, member := range r.byID {
		if member.RoomID == roomID {
			member.IsDeleted = true
			member.DeletedAt = &now
			member.UpdatedAt = now
		}
	}
	return nil
}

func (r *bddChatMemberRepo) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type bddChatMessageRepo struct {
	mu     sync.Mutex
	nextID chatmessage.ID
	byRoom map[chatroom.ID][]*chatmessage.ChatMessage
}

func newBDDChatMessageRepo() *bddChatMessageRepo {
	return &bddChatMessageRepo{}
}

func (r *bddChatMessageRepo) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID = 1
	r.byRoom = map[chatroom.ID][]*chatmessage.ChatMessage{}
}

func (r *bddChatMessageRepo) FindByID(context.Context, chatmessage.ID) (*chatmessage.ChatMessage, error) {
	return nil, chatmessage.ErrNotFound
}

func (r *bddChatMessageRepo) FindByRoom(_ context.Context, roomID chatroom.ID, limit, offset uint64) ([]*chatmessage.ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	messages := r.byRoom[roomID]
	if offset >= uint64(len(messages)) {
		return nil, nil
	}
	end := uint64(len(messages))
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	out := make([]*chatmessage.ChatMessage, 0, end-offset)
	for _, msg := range messages[offset:end] {
		copied := *msg
		out = append(out, &copied)
	}
	return out, nil
}

func (r *bddChatMessageRepo) FindByRoomBefore(context.Context, chatroom.ID, chatmessage.ID, uint64) ([]*chatmessage.ChatMessage, error) {
	return nil, nil
}

func (r *bddChatMessageRepo) FindLatestByRoom(_ context.Context, roomID chatroom.ID) (*chatmessage.ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	messages := r.byRoom[roomID]
	if len(messages) == 0 {
		return nil, chatmessage.ErrNotFound
	}
	latest := messages[len(messages)-1]
	copied := *latest
	return &copied, nil
}

func (r *bddChatMessageRepo) CountByRoomAfter(_ context.Context, roomID chatroom.ID, since time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for _, msg := range r.byRoom[roomID] {
		if !msg.IsDeleted && msg.CreatedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (r *bddChatMessageRepo) FindBySender(context.Context, chatmember.ID) ([]*chatmessage.ChatMessage, error) {
	return nil, nil
}

func (r *bddChatMessageRepo) Create(_ context.Context, msg *chatmessage.ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg.ID == 0 {
		msg.ID = r.nextID
		r.nextID++
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	copied := *msg
	r.byRoom[msg.RoomID] = append(r.byRoom[msg.RoomID], &copied)
	return nil
}

func (r *bddChatMessageRepo) Update(context.Context, *chatmessage.ChatMessage) error {
	return nil
}

func (r *bddChatMessageRepo) SoftDelete(context.Context, chatmessage.ID) error {
	return nil
}

type bddAgentRepo struct{}

func (r *bddAgentRepo) FindByID(context.Context, agent.ID) (*agent.Agent, error) {
	return nil, agent.ErrNotFound
}

func (r *bddAgentRepo) FindAll(context.Context) ([]*agent.Agent, error) {
	return nil, nil
}

func (r *bddAgentRepo) FindAllByStatus(context.Context, agent.Status) ([]*agent.Agent, error) {
	return nil, nil
}

func (r *bddAgentRepo) Create(context.Context, *agent.Agent) error {
	return nil
}

type bddBroadcaster struct{}

func (bddBroadcaster) SendToUser(string, []byte) {}

type bddFileStorage struct{}

func (bddFileStorage) Save(context.Context, shared.File, string) (appPort.SaveResult, error) {
	return appPort.SaveResult{}, nil
}

func (bddFileStorage) SaveStream(context.Context, io.Reader, appPort.FileMeta, string) (appPort.SaveResult, error) {
	return appPort.SaveResult{}, nil
}

func (bddFileStorage) Delete(context.Context, string) error {
	return nil
}

func (bddFileStorage) URL(path string) string {
	return path
}

var (
	_ domainuser.Repository  = (*bddUserRepo)(nil)
	_ participant.Repository = (*bddParticipantRepo)(nil)
	_ chatroom.Repository    = (*bddChatRoomRepo)(nil)
	_ chatmember.Repository  = (*bddChatMemberRepo)(nil)
	_ chatmessage.Repository = (*bddChatMessageRepo)(nil)
	_ agent.Repository       = (*bddAgentRepo)(nil)
)
