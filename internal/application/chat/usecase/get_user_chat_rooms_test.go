package usecase

import (
	"context"
	"testing"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserChatRoomsUseCase_IncludesLatestMessageSenderAndDirectAvatar(t *testing.T) {
	now := time.Now()
	callerUserID := shared.UserID(10)
	otherUserID := shared.UserID(20)
	callerParticipantID := participant.ID(1)
	otherParticipantID := participant.ID(2)
	directRoomID := chatroom.ID(7)
	groupRoomID := chatroom.ID(8)
	otherMemberID := chatmember.ID(101)

	participantRepo := &getRoomsParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherUserID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherParticipantID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
	}
	memberRepo := &getRoomsChatMemberRepoStub{
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			callerParticipantID: {
				{ID: 100, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
				{ID: 200, RoomID: groupRoomID, ParticipantID: callerParticipantID, Role: chatmember.Member, JoinedAt: now.Add(-time.Hour)},
			},
		},
		byRoom: map[chatroom.ID][]*chatmember.ChatMember{
			directRoomID: {
				{ID: 100, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
				{ID: otherMemberID, RoomID: directRoomID, ParticipantID: otherParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
			},
			groupRoomID: {
				{ID: 200, RoomID: groupRoomID, ParticipantID: callerParticipantID, Role: chatmember.Member, JoinedAt: now.Add(-time.Hour)},
			},
		},
	}
	roomRepo := &getRoomsChatRoomRepoStub{
		rooms: map[chatroom.ID]*chatroom.ChatRoom{
			directRoomID: {ID: directRoomID, Name: "Direct fallback", Type: chatroom.Direct},
			groupRoomID:  {ID: groupRoomID, Name: "Launch Crew", Type: chatroom.Group},
		},
	}
	messageRepo := &getRoomsChatMessageRepoStub{
		latestByRoom: map[chatroom.ID]*chatmessage.ChatMessage{
			directRoomID: {
				ID:        301,
				RoomID:    directRoomID,
				SenderID:  otherMemberID,
				Content:   "e2ee:v1:ciphertext",
				Type:      chatmessage.Text,
				CreatedAt: now,
			},
		},
		unreadByRoom: map[chatroom.ID]int64{
			directRoomID: 1,
			groupRoomID:  0,
		},
	}
	userRepo := &getRoomsUserRepoStub{
		byID: map[shared.UserID]*domainuser.User{
			otherUserID: {ID: otherUserID, Name: "Luna", Avatar: "avatars/luna.png"},
		},
	}

	uc := NewGetUserChatRoomsUseCase(
		participantRepo,
		memberRepo,
		roomRepo,
		messageRepo,
		userRepo,
		&getRoomsAgentRepoStub{},
		nil,
		nil,
	)

	t.Log("Given: a user belongs to a direct room with an encrypted latest message and a group room with no messages")
	t.Logf("Input: caller_user_id=%d direct_room_id=%d latest_sender_member_id=%d", callerUserID, directRoomID, otherMemberID)
	t.Log("Action: execute GetUserChatRoomsUseCase")

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[struct{}]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
	})

	t.Logf("Output: direct_count=%d group_count=%d", len(out.Direct), len(out.Group))
	t.Log("Mutation: none")
	require.NoError(t, err)
	require.Len(t, out.Direct, 1)
	require.Len(t, out.Group, 1)

	direct := out.Direct[0]
	require.NotNil(t, direct.LatestMsg)
	require.NotNil(t, direct.LatestMsgCreatedAt)
	require.NotNil(t, direct.LatestMsgSenderID)
	assert.Equal(t, "Luna", direct.DisplayName)
	assert.Equal(t, "avatars/luna.png", *direct.AvatarURL)
	assert.Equal(t, "e2ee:v1:ciphertext", *direct.LatestMsg)
	assert.WithinDuration(t, now, *direct.LatestMsgCreatedAt, time.Second)
	assert.Equal(t, int64(otherMemberID), *direct.LatestMsgSenderID)
	assert.Equal(t, int64(1), direct.UnreadCount)

	group := out.Group[0]
	assert.Equal(t, "Launch Crew", group.DisplayName)
	assert.Nil(t, group.LatestMsg)
	assert.Nil(t, group.LatestMsgCreatedAt)
	assert.Nil(t, group.LatestMsgSenderID)
}

func TestGetUserChatRoomsUseCase_IncludesDirectPeerPresence(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	callerUserID := shared.UserID(10)
	otherUserID := shared.UserID(20)
	callerParticipantID := participant.ID(1)
	otherParticipantID := participant.ID(2)
	directRoomID := chatroom.ID(7)

	participantRepo := &getRoomsParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherUserID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherParticipantID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
	}
	memberRepo := &getRoomsChatMemberRepoStub{
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			callerParticipantID: {
				{ID: 100, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
			},
		},
		byRoom: map[chatroom.ID][]*chatmember.ChatMember{
			directRoomID: {
				{ID: 100, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
				{ID: 101, RoomID: directRoomID, ParticipantID: otherParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
			},
		},
	}
	roomRepo := &getRoomsChatRoomRepoStub{
		rooms: map[chatroom.ID]*chatroom.ChatRoom{
			directRoomID: {ID: directRoomID, Name: "Direct fallback", Type: chatroom.Direct},
		},
	}
	userRepo := &getRoomsUserRepoStub{
		byID: map[shared.UserID]*domainuser.User{
			otherUserID: {ID: otherUserID, Name: "Luna"},
		},
	}
	presenceReader := getRoomsPresenceReaderStub{
		snapshots: map[string]chatPort.PresenceSnapshot{
			"20": {
				Status:     chatPort.PresenceStatusOffline,
				LastSeenAt: &now,
			},
		},
	}

	uc := NewGetUserChatRoomsUseCase(
		participantRepo,
		memberRepo,
		roomRepo,
		&getRoomsChatMessageRepoStub{},
		userRepo,
		&getRoomsAgentRepoStub{},
		nil,
		presenceReader,
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[struct{}]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
	})

	require.NoError(t, err)
	require.Len(t, out.Direct, 1)
	assert.Equal(t, int64(otherUserID), *out.Direct[0].PeerUserID)
	require.NotNil(t, out.Direct[0].PresenceStatus)
	assert.Equal(t, string(chatPort.PresenceStatusOffline), *out.Direct[0].PresenceStatus)
	require.NotNil(t, out.Direct[0].LastSeenAt)
	assert.True(t, out.Direct[0].LastSeenAt.Equal(now))
}

func TestGetUserChatRoomsUseCase_ExcludesCallerMessagesFromUnreadCount(t *testing.T) {
	now := time.Now()
	callerUserID := shared.UserID(10)
	otherUserID := shared.UserID(20)
	callerParticipantID := participant.ID(1)
	otherParticipantID := participant.ID(2)
	directRoomID := chatroom.ID(7)
	callerMemberID := chatmember.ID(100)

	participantRepo := &getRoomsParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherUserID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			otherParticipantID:  {ID: otherParticipantID, Type: participant.UserType, UserID: &otherUserID},
		},
	}
	memberRepo := &getRoomsChatMemberRepoStub{
		byParticipant: map[participant.ID][]*chatmember.ChatMember{
			callerParticipantID: {
				{ID: callerMemberID, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
			},
		},
		byRoom: map[chatroom.ID][]*chatmember.ChatMember{
			directRoomID: {
				{ID: callerMemberID, RoomID: directRoomID, ParticipantID: callerParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
				{ID: 101, RoomID: directRoomID, ParticipantID: otherParticipantID, Role: chatmember.Owner, JoinedAt: now.Add(-time.Hour)},
			},
		},
	}
	messageRepo := &getRoomsChatMessageRepoStub{
		unreadByRoom: map[chatroom.ID]int64{
			directRoomID: 2,
		},
	}
	uc := NewGetUserChatRoomsUseCase(
		participantRepo,
		memberRepo,
		&getRoomsChatRoomRepoStub{
			rooms: map[chatroom.ID]*chatroom.ChatRoom{
				directRoomID: {ID: directRoomID, Name: "Direct", Type: chatroom.Direct},
			},
		},
		messageRepo,
		&getRoomsUserRepoStub{
			byID: map[shared.UserID]*domainuser.User{
				otherUserID: {ID: otherUserID, Name: "Luna"},
			},
		},
		&getRoomsAgentRepoStub{},
		nil,
		nil,
	)

	_, err := uc.Execute(context.Background(), appShared.UseCaseInput[struct{}]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
	})

	require.NoError(t, err)
	assert.Contains(t, messageRepo.lastExcludedSenderIDs, callerMemberID)
}

type getRoomsParticipantRepoStub struct {
	byUser map[shared.UserID]*participant.Participant
	byID   map[participant.ID]*participant.Participant
}

func (s *getRoomsParticipantRepoStub) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	if p, ok := s.byID[id]; ok {
		copied := *p
		return &copied, nil
	}
	return nil, participant.ErrNotFound
}

func (s *getRoomsParticipantRepoStub) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	if p, ok := s.byUser[userID]; ok {
		copied := *p
		return &copied, nil
	}
	return nil, participant.ErrNotFound
}

func (s *getRoomsParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *getRoomsParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *getRoomsParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type getRoomsChatMemberRepoStub struct {
	byParticipant map[participant.ID][]*chatmember.ChatMember
	byRoom        map[chatroom.ID][]*chatmember.ChatMember
}

func (s *getRoomsChatMemberRepoStub) FindByID(_ context.Context, id chatmember.ID) (*chatmember.ChatMember, error) {
	for _, members := range s.byRoom {
		for _, m := range members {
			if m.ID == id {
				copied := *m
				return &copied, nil
			}
		}
	}
	return nil, chatmember.ErrNotFound
}

func (s *getRoomsChatMemberRepoStub) FindByRoomAndParticipant(_ context.Context, roomID chatroom.ID, participantID participant.ID) (*chatmember.ChatMember, error) {
	for _, m := range s.byRoom[roomID] {
		if m.ParticipantID == participantID && !m.IsDeleted {
			copied := *m
			return &copied, nil
		}
	}
	return nil, chatmember.ErrNotFound
}

func (s *getRoomsChatMemberRepoStub) FindByRoom(_ context.Context, roomID chatroom.ID) ([]*chatmember.ChatMember, error) {
	return cloneGetRoomsMembers(s.byRoom[roomID]), nil
}

func (s *getRoomsChatMemberRepoStub) FindByParticipant(_ context.Context, participantID participant.ID) ([]*chatmember.ChatMember, error) {
	return cloneGetRoomsMembers(s.byParticipant[participantID]), nil
}

func (s *getRoomsChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *getRoomsChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *getRoomsChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *getRoomsChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

func cloneGetRoomsMembers(items []*chatmember.ChatMember) []*chatmember.ChatMember {
	out := make([]*chatmember.ChatMember, 0, len(items))
	for _, item := range items {
		copied := *item
		out = append(out, &copied)
	}
	return out
}

type getRoomsChatRoomRepoStub struct {
	rooms map[chatroom.ID]*chatroom.ChatRoom
}

func (s *getRoomsChatRoomRepoStub) FindByID(_ context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	if room, ok := s.rooms[id]; ok {
		copied := *room
		return &copied, nil
	}
	return nil, chatroom.ErrNotFound
}

func (s *getRoomsChatRoomRepoStub) Create(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (s *getRoomsChatRoomRepoStub) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}

func (s *getRoomsChatRoomRepoStub) Update(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (s *getRoomsChatRoomRepoStub) SoftDelete(context.Context, chatroom.ID) error {
	return nil
}

type getRoomsChatMessageRepoStub struct {
	latestByRoom          map[chatroom.ID]*chatmessage.ChatMessage
	unreadByRoom          map[chatroom.ID]int64
	lastExcludedSenderIDs []chatmember.ID
}

func (s *getRoomsChatMessageRepoStub) FindByID(context.Context, chatmessage.ID) (*chatmessage.ChatMessage, error) {
	return nil, chatmessage.ErrNotFound
}

func (s *getRoomsChatMessageRepoStub) FindByRoom(context.Context, chatroom.ID, uint64, uint64) ([]*chatmessage.ChatMessage, error) {
	return nil, nil
}

func (s *getRoomsChatMessageRepoStub) FindByRoomBefore(context.Context, chatroom.ID, chatmessage.ID, uint64) ([]*chatmessage.ChatMessage, error) {
	return nil, nil
}

func (s *getRoomsChatMessageRepoStub) FindLatestByRoom(_ context.Context, roomID chatroom.ID) (*chatmessage.ChatMessage, error) {
	if msg, ok := s.latestByRoom[roomID]; ok {
		copied := *msg
		return &copied, nil
	}
	return nil, chatmessage.ErrNotFound
}

func (s *getRoomsChatMessageRepoStub) CountByRoomAfter(_ context.Context, roomID chatroom.ID, _ time.Time) (int64, error) {
	return s.unreadByRoom[roomID], nil
}

func (s *getRoomsChatMessageRepoStub) FindByRoomExcludingSenders(_ context.Context, roomID chatroom.ID, _ []chatmember.ID, _ uint64, _ uint64) ([]*chatmessage.ChatMessage, error) {
	return s.FindByRoom(context.Background(), roomID, 0, 0)
}

func (s *getRoomsChatMessageRepoStub) FindByRoomBeforeExcludingSenders(_ context.Context, roomID chatroom.ID, _ chatmessage.ID, _ []chatmember.ID, _ uint64) ([]*chatmessage.ChatMessage, error) {
	return s.FindByRoom(context.Background(), roomID, 0, 0)
}

func (s *getRoomsChatMessageRepoStub) FindLatestByRoomExcludingSenders(_ context.Context, roomID chatroom.ID, _ []chatmember.ID) (*chatmessage.ChatMessage, error) {
	return s.FindLatestByRoom(context.Background(), roomID)
}

func (s *getRoomsChatMessageRepoStub) CountByRoomAfterExcludingSenders(_ context.Context, roomID chatroom.ID, _ time.Time, excludedSenderIDs []chatmember.ID) (int64, error) {
	s.lastExcludedSenderIDs = append([]chatmember.ID(nil), excludedSenderIDs...)
	return s.unreadByRoom[roomID], nil
}

func (s *getRoomsChatMessageRepoStub) FindBySender(context.Context, chatmember.ID) ([]*chatmessage.ChatMessage, error) {
	return nil, nil
}

func (s *getRoomsChatMessageRepoStub) Create(context.Context, *chatmessage.ChatMessage) error {
	return nil
}

func (s *getRoomsChatMessageRepoStub) Update(context.Context, *chatmessage.ChatMessage) error {
	return nil
}

func (s *getRoomsChatMessageRepoStub) SoftDelete(context.Context, chatmessage.ID) error {
	return nil
}

type getRoomsUserRepoStub struct {
	byID map[shared.UserID]*domainuser.User
}

func (s *getRoomsUserRepoStub) Create(context.Context, *domainuser.User) (shared.UserID, error) {
	return 0, nil
}

func (s *getRoomsUserRepoStub) FindByID(_ context.Context, id shared.UserID) (*domainuser.User, error) {
	if user, ok := s.byID[id]; ok {
		copied := *user
		return &copied, nil
	}
	return nil, domainuser.ErrUserNotFound
}

func (s *getRoomsUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]domainuser.User, error) {
	return nil, nil
}

func (s *getRoomsUserRepoStub) Update(context.Context, *domainuser.User) error {
	return nil
}

func (s *getRoomsUserRepoStub) SearchByName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (s *getRoomsUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

func (s *getRoomsUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*domainuser.UserSearchResult, error) {
	return nil, nil
}

type getRoomsAgentRepoStub struct{}

type getRoomsPresenceReaderStub struct {
	snapshots map[string]chatPort.PresenceSnapshot
}

func (s getRoomsPresenceReaderStub) GetPresence(userID string) chatPort.PresenceSnapshot {
	if snapshot, ok := s.snapshots[userID]; ok {
		return snapshot
	}
	return chatPort.PresenceSnapshot{Status: chatPort.PresenceStatusOffline}
}

func (s *getRoomsAgentRepoStub) FindByID(context.Context, agent.ID) (*agent.Agent, error) {
	return nil, agent.ErrNotFound
}

func (s *getRoomsAgentRepoStub) FindAll(context.Context) ([]*agent.Agent, error) {
	return nil, nil
}

func (s *getRoomsAgentRepoStub) FindAllByStatus(context.Context, agent.Status) ([]*agent.Agent, error) {
	return nil, nil
}

func (s *getRoomsAgentRepoStub) Create(context.Context, *agent.Agent) error {
	return nil
}
