package usecase

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	appPort "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainUser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/require"
)

func TestDeletedRoomUseCasesRejectRoomBeforeMemberSideEffects(t *testing.T) {
	ctx := context.Background()
	callerUserID := shared.UserID(10)
	callerParticipantID := participant.ID(20)
	roomID := chatroom.ID(30)

	participantRepo := &deletedRoomParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
		},
	}
	roomRepo := &deletedRoomChatRoomRepoStub{
		rooms: map[chatroom.ID]*chatroom.ChatRoom{
			roomID: {ID: roomID, Type: chatroom.Direct, IsDeleted: true},
		},
	}
	memberRepo := &deletedRoomChatMemberRepoStub{}
	messageRepo := &deletedRoomChatMessageRepoStub{}
	fileStorage := &deletedRoomFileStorageStub{}

	base := appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}}

	t.Log("Given: a deleted direct chat room exists for the caller")
	t.Logf("Input: caller_user_id=%d room_id=%d is_deleted=true", callerUserID, roomID)

	t.Run("detail", func(t *testing.T) {
		t.Log("Action: execute GetChatRoomDetailUseCase")

		_, err := NewGetChatRoomDetailUseCase(
			participantRepo,
			memberRepo,
			roomRepo,
			messageRepo,
			&deletedRoomUserRepoStub{},
			&deletedRoomAgentRepoStub{},
		).Execute(ctx, appShared.UseCaseInput[GetChatRoomDetailInput]{
			Base: base,
			Data: GetChatRoomDetailInput{RoomID: int64(roomID)},
		})

		t.Logf("Output: err=%v", err)
		t.Log("Mutation: no member lookup or message fetch")
		require.ErrorIs(t, err, ErrChatRoomNotFound)
	})

	t.Run("messages", func(t *testing.T) {
		t.Log("Action: execute GetChatRoomMessagesUseCase")

		_, err := NewGetChatRoomMessagesUseCase(
			participantRepo,
			memberRepo,
			roomRepo,
			messageRepo,
		).Execute(ctx, appShared.UseCaseInput[GetChatRoomMessagesInput]{
			Base: base,
			Data: GetChatRoomMessagesInput{RoomID: int64(roomID), Limit: 20},
		})

		t.Logf("Output: err=%v", err)
		t.Log("Mutation: no member lookup or message fetch")
		require.ErrorIs(t, err, ErrChatRoomNotFound)
	})

	t.Run("send", func(t *testing.T) {
		t.Log("Action: execute SendMessageUseCase")

		_, err := NewSendMessageUseCase(
			participantRepo,
			memberRepo,
			roomRepo,
			messageRepo,
			deletedRoomBroadcasterStub{},
		).Execute(ctx, appShared.UseCaseInput[SendMessageInput]{
			Base: base,
			Data: SendMessageInput{RoomID: int64(roomID), Content: "hello", Type: string(chatmessage.Text)},
		})

		t.Logf("Output: err=%v", err)
		t.Log("Mutation: message not created")
		require.ErrorIs(t, err, ErrChatRoomNotFound)
	})

	t.Run("mark read", func(t *testing.T) {
		t.Log("Action: execute UpdateMemberStatusUseCase")

		_, err := NewUpdateMemberStatusUseCase(
			participantRepo,
			memberRepo,
			roomRepo,
		).Execute(ctx, appShared.UseCaseInput[UpdateMemberStatusInput]{
			Base: base,
			Data: UpdateMemberStatusInput{RoomID: int64(roomID)},
		})

		t.Logf("Output: err=%v", err)
		t.Log("Mutation: member status not updated")
		require.ErrorIs(t, err, ErrChatRoomNotFound)
	})

	t.Run("upload", func(t *testing.T) {
		t.Log("Action: execute UploadRoomMediaUseCase")

		_, err := NewUploadRoomMediaUseCase(
			participantRepo,
			memberRepo,
			roomRepo,
			fileStorage,
		).Execute(ctx, appShared.UseCaseInput[UploadRoomMediaInput]{
			Base: base,
			Data: UploadRoomMediaInput{
				RoomID:   int64(roomID),
				File:     bytes.NewBufferString("file"),
				Filename: "note.txt",
				MimeType: "text/plain",
				Size:     4,
			},
		})

		t.Logf("Output: err=%v", err)
		t.Log("Mutation: file not stored")
		require.ErrorIs(t, err, ErrChatRoomNotFound)
	})

	require.Zero(t, memberRepo.findMemberCalls)
	require.Zero(t, memberRepo.updateCalls)
	require.Zero(t, messageRepo.findCalls)
	require.Zero(t, messageRepo.createCalls)
	require.Zero(t, fileStorage.saveCalls)
}

type deletedRoomParticipantRepoStub struct {
	byUser map[shared.UserID]*participant.Participant
}

func (s *deletedRoomParticipantRepoStub) FindByID(_ context.Context, id participant.ID) (*participant.Participant, error) {
	for _, p := range s.byUser {
		if p.ID == id {
			copied := *p
			return &copied, nil
		}
	}
	return nil, participant.ErrNotFound
}

func (s *deletedRoomParticipantRepoStub) FindByUserID(_ context.Context, userID shared.UserID) (*participant.Participant, error) {
	if p, ok := s.byUser[userID]; ok {
		copied := *p
		return &copied, nil
	}
	return nil, participant.ErrNotFound
}

func (s *deletedRoomParticipantRepoStub) FindByAgentID(context.Context, int64) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *deletedRoomParticipantRepoStub) FindSystemByType(context.Context, string) (*participant.Participant, error) {
	return nil, participant.ErrNotFound
}

func (s *deletedRoomParticipantRepoStub) Create(context.Context, *participant.Participant) error {
	return nil
}

type deletedRoomChatRoomRepoStub struct {
	rooms map[chatroom.ID]*chatroom.ChatRoom
}

func (s *deletedRoomChatRoomRepoStub) FindByID(_ context.Context, id chatroom.ID) (*chatroom.ChatRoom, error) {
	if room, ok := s.rooms[id]; ok {
		copied := *room
		return &copied, nil
	}
	return nil, chatroom.ErrNotFound
}

func (s *deletedRoomChatRoomRepoStub) Create(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (s *deletedRoomChatRoomRepoStub) FindDirectByParticipants(context.Context, participant.ID, participant.ID) (*chatroom.ChatRoom, error) {
	return nil, chatroom.ErrNotFound
}

func (s *deletedRoomChatRoomRepoStub) Update(context.Context, *chatroom.ChatRoom) error {
	return nil
}

func (s *deletedRoomChatRoomRepoStub) SoftDelete(context.Context, chatroom.ID) error {
	return nil
}

type deletedRoomChatMemberRepoStub struct {
	findMemberCalls int
	updateCalls     int
}

func (s *deletedRoomChatMemberRepoStub) FindByID(context.Context, chatmember.ID) (*chatmember.ChatMember, error) {
	s.findMemberCalls++
	return nil, chatmember.ErrNotFound
}

func (s *deletedRoomChatMemberRepoStub) FindByRoomAndParticipant(context.Context, chatroom.ID, participant.ID) (*chatmember.ChatMember, error) {
	s.findMemberCalls++
	return nil, chatmember.ErrNotFound
}

func (s *deletedRoomChatMemberRepoStub) FindByRoom(context.Context, chatroom.ID) ([]*chatmember.ChatMember, error) {
	s.findMemberCalls++
	return nil, nil
}

func (s *deletedRoomChatMemberRepoStub) FindByParticipant(context.Context, participant.ID) ([]*chatmember.ChatMember, error) {
	s.findMemberCalls++
	return nil, nil
}

func (s *deletedRoomChatMemberRepoStub) Add(context.Context, *chatmember.ChatMember) error {
	return nil
}

func (s *deletedRoomChatMemberRepoStub) Update(context.Context, *chatmember.ChatMember) error {
	s.updateCalls++
	return nil
}

func (s *deletedRoomChatMemberRepoStub) SoftDelete(context.Context, chatmember.ID) error {
	return nil
}

func (s *deletedRoomChatMemberRepoStub) Remove(context.Context, chatroom.ID, participant.ID) error {
	return nil
}

type deletedRoomChatMessageRepoStub struct {
	findCalls   int
	createCalls int
}

func (s *deletedRoomChatMessageRepoStub) FindByID(context.Context, chatmessage.ID) (*chatmessage.ChatMessage, error) {
	s.findCalls++
	return nil, chatmessage.ErrNotFound
}

func (s *deletedRoomChatMessageRepoStub) FindByRoom(context.Context, chatroom.ID, uint64, uint64) ([]*chatmessage.ChatMessage, error) {
	s.findCalls++
	return nil, nil
}

func (s *deletedRoomChatMessageRepoStub) FindByRoomBefore(context.Context, chatroom.ID, chatmessage.ID, uint64) ([]*chatmessage.ChatMessage, error) {
	s.findCalls++
	return nil, nil
}

func (s *deletedRoomChatMessageRepoStub) FindLatestByRoom(context.Context, chatroom.ID) (*chatmessage.ChatMessage, error) {
	s.findCalls++
	return nil, chatmessage.ErrNotFound
}

func (s *deletedRoomChatMessageRepoStub) CountByRoomAfter(context.Context, chatroom.ID, time.Time) (int64, error) {
	s.findCalls++
	return 0, nil
}

func (s *deletedRoomChatMessageRepoStub) FindBySender(context.Context, chatmember.ID) ([]*chatmessage.ChatMessage, error) {
	s.findCalls++
	return nil, nil
}

func (s *deletedRoomChatMessageRepoStub) Create(context.Context, *chatmessage.ChatMessage) error {
	s.createCalls++
	return nil
}

func (s *deletedRoomChatMessageRepoStub) Update(context.Context, *chatmessage.ChatMessage) error {
	return nil
}

func (s *deletedRoomChatMessageRepoStub) SoftDelete(context.Context, chatmessage.ID) error {
	return nil
}

type deletedRoomFileStorageStub struct {
	saveCalls int
}

func (s *deletedRoomFileStorageStub) Save(context.Context, shared.File, string) (appPort.SaveResult, error) {
	s.saveCalls++
	return appPort.SaveResult{}, nil
}

func (s *deletedRoomFileStorageStub) SaveStream(context.Context, io.Reader, appPort.FileMeta, string) (appPort.SaveResult, error) {
	s.saveCalls++
	return appPort.SaveResult{}, nil
}

func (s *deletedRoomFileStorageStub) Delete(context.Context, string) error {
	return nil
}

func (s *deletedRoomFileStorageStub) URL(path string) string {
	return path
}

type deletedRoomUserRepoStub struct{}

func (deletedRoomUserRepoStub) Create(context.Context, *domainUser.User) (shared.UserID, error) {
	return 0, nil
}

func (deletedRoomUserRepoStub) FindByID(context.Context, shared.UserID) (*domainUser.User, error) {
	return nil, domainUser.ErrUserNotFound
}

func (deletedRoomUserRepoStub) FindByAccountID(context.Context, shared.AccountID) (*[]domainUser.User, error) {
	return nil, domainUser.ErrUserNotFound
}

func (deletedRoomUserRepoStub) Update(context.Context, *domainUser.User) error {
	return nil
}

func (deletedRoomUserRepoStub) SearchByName(context.Context, string, int, int) ([]*domainUser.UserSearchResult, error) {
	return nil, nil
}

func (deletedRoomUserRepoStub) FindByAccountName(context.Context, string, int, int) ([]*domainUser.UserSearchResult, error) {
	return nil, nil
}

func (deletedRoomUserRepoStub) FindByPublicID(context.Context, string, int, int) ([]*domainUser.UserSearchResult, error) {
	return nil, nil
}

type deletedRoomAgentRepoStub struct{}

func (deletedRoomAgentRepoStub) FindByID(context.Context, agent.ID) (*agent.Agent, error) {
	return nil, agent.ErrNotFound
}

func (deletedRoomAgentRepoStub) FindAll(context.Context) ([]*agent.Agent, error) {
	return nil, nil
}

func (deletedRoomAgentRepoStub) FindAllByStatus(context.Context, agent.Status) ([]*agent.Agent, error) {
	return nil, nil
}

func (deletedRoomAgentRepoStub) Create(context.Context, *agent.Agent) error {
	return nil
}

type deletedRoomBroadcasterStub struct{}

func (deletedRoomBroadcasterStub) SendToUser(string, []byte) {}
