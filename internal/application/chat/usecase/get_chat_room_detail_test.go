package usecase

import (
	"context"
	"testing"
	"time"

	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	domainuser "github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetChatRoomDetailUseCase_ResolvesDirectPeerDisplayName(t *testing.T) {
	now := time.Now()
	callerUserID := shared.UserID(10)
	peerUserID := shared.UserID(20)
	callerParticipantID := participant.ID(100)
	peerParticipantID := participant.ID(200)
	roomID := chatroom.ID(300)

	participantRepo := &getRoomsParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			peerUserID:   {ID: peerParticipantID, Type: participant.UserType, UserID: &peerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			peerParticipantID:   {ID: peerParticipantID, Type: participant.UserType, UserID: &peerUserID},
		},
	}
	memberRepo := &getRoomsChatMemberRepoStub{
		byRoom: map[chatroom.ID][]*chatmember.ChatMember{
			roomID: {
				{ID: 1, RoomID: roomID, ParticipantID: callerParticipantID, JoinedAt: now.Add(-time.Hour)},
				{ID: 2, RoomID: roomID, ParticipantID: peerParticipantID, JoinedAt: now.Add(-time.Hour)},
			},
		},
	}
	roomRepo := &getRoomsChatRoomRepoStub{
		rooms: map[chatroom.ID]*chatroom.ChatRoom{
			roomID: {ID: roomID, Name: "", Type: chatroom.Direct},
		},
	}
	userRepo := &getRoomsUserRepoStub{
		byID: map[shared.UserID]*domainuser.User{
			callerUserID: {ID: callerUserID, Name: "Hiro"},
			peerUserID:   {ID: peerUserID, Name: "Mizi Liang", Avatar: "avatars/mizi.png"},
		},
	}

	uc := NewGetChatRoomDetailUseCase(
		participantRepo,
		memberRepo,
		roomRepo,
		&getRoomsChatMessageRepoStub{},
		userRepo,
		&getRoomsAgentRepoStub{},
		nil,
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetChatRoomDetailInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
		Data: GetChatRoomDetailInput{RoomID: int64(roomID)},
	})

	require.NoError(t, err)
	assert.Equal(t, "Mizi Liang", out.Name)
	require.NotNil(t, out.AvatarURL)
	assert.Equal(t, "avatars/mizi.png", *out.AvatarURL)
}

func TestGetChatRoomDetailUseCase_ResolvesBotDisplayName(t *testing.T) {
	now := time.Now()
	callerUserID := shared.UserID(10)
	callerParticipantID := participant.ID(100)
	agentID := int64(700)
	botParticipantID := participant.ID(200)
	roomID := chatroom.ID(301)

	participantRepo := &getRoomsParticipantRepoStub{
		byUser: map[shared.UserID]*participant.Participant{
			callerUserID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
		},
		byID: map[participant.ID]*participant.Participant{
			callerParticipantID: {ID: callerParticipantID, Type: participant.UserType, UserID: &callerUserID},
			botParticipantID:    {ID: botParticipantID, Type: participant.AgentType, AgentID: &agentID},
		},
	}
	memberRepo := &getRoomsChatMemberRepoStub{
		byRoom: map[chatroom.ID][]*chatmember.ChatMember{
			roomID: {
				{ID: 1, RoomID: roomID, ParticipantID: callerParticipantID, JoinedAt: now.Add(-time.Hour)},
				{ID: 2, RoomID: roomID, ParticipantID: botParticipantID, JoinedAt: now.Add(-time.Hour)},
			},
		},
	}
	roomRepo := &getRoomsChatRoomRepoStub{
		rooms: map[chatroom.ID]*chatroom.ChatRoom{
			roomID: {ID: roomID, Name: "", Type: chatroom.Bot},
		},
	}

	uc := NewGetChatRoomDetailUseCase(
		participantRepo,
		memberRepo,
		roomRepo,
		&getRoomsChatMessageRepoStub{},
		&getRoomsUserRepoStub{},
		&detailAgentRepoStub{agents: map[agent.ID]*agent.Agent{
			agent.ID(agentID): {ID: agent.ID(agentID), Name: "Support Bot"},
		}},
		nil,
	)

	out, err := uc.Execute(context.Background(), appShared.UseCaseInput[GetChatRoomDetailInput]{
		Base: appShared.BaseContext{Auth: &appShared.AuthContext{UserID: callerUserID}},
		Data: GetChatRoomDetailInput{RoomID: int64(roomID)},
	})

	require.NoError(t, err)
	assert.Equal(t, "Support Bot", out.Name)
}

type detailAgentRepoStub struct {
	agents map[agent.ID]*agent.Agent
}

func (s *detailAgentRepoStub) FindByID(_ context.Context, id agent.ID) (*agent.Agent, error) {
	if item, ok := s.agents[id]; ok {
		copied := *item
		return &copied, nil
	}
	return nil, agent.ErrNotFound
}

func (s *detailAgentRepoStub) FindAll(context.Context) ([]*agent.Agent, error) {
	return nil, nil
}

func (s *detailAgentRepoStub) FindAllByStatus(context.Context, agent.Status) ([]*agent.Agent, error) {
	return nil, nil
}

func (s *detailAgentRepoStub) Create(context.Context, *agent.Agent) error {
	return nil
}
