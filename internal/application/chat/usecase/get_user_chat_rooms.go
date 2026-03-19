package usecase

import (
	"context"
	"errors"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type ChatRoomSummary struct {
	RoomID      int64
	RoomType    string
	DisplayName string
	AvatarURL   *string
	LatestMsg   *string
	UnreadCount int64
}

type GetUserChatRoomsOutput struct {
	Direct  []ChatRoomSummary
	Group   []ChatRoomSummary
	Channel []ChatRoomSummary
	Bot     []ChatRoomSummary
}

type GetUserChatRoomsUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	chatRoomRepo    chatroom.Repository
	chatMessageRepo chatmessage.Repository
	userRepo        user.Repository
	agentRepo       agent.Repository
}

func NewGetUserChatRoomsUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatRoomRepo chatroom.Repository,
	chatMessageRepo chatmessage.Repository,
	userRepo user.Repository,
	agentRepo agent.Repository,
) *GetUserChatRoomsUseCase {
	return &GetUserChatRoomsUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMessageRepo: chatMessageRepo,
		userRepo:        userRepo,
		agentRepo:       agentRepo,
	}
}

func (uc *GetUserChatRoomsUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[struct{}],
) (GetUserChatRoomsOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetUserChatRoomsOutput{}, ErrParticipantNotFound
		}
		return GetUserChatRoomsOutput{}, err
	}

	members, err := uc.chatMemberRepo.FindByParticipant(ctx, callerParticipant.ID)
	if err != nil {
		return GetUserChatRoomsOutput{}, err
	}

	out := GetUserChatRoomsOutput{
		Direct:  []ChatRoomSummary{},
		Group:   []ChatRoomSummary{},
		Channel: []ChatRoomSummary{},
		Bot:     []ChatRoomSummary{},
	}

	for _, member := range members {
		if member.IsDeleted {
			continue
		}

		room, err := uc.chatRoomRepo.FindByID(ctx, member.RoomID)
		if err != nil || room.IsDeleted {
			continue
		}

		displayName := uc.resolveDisplayName(ctx, room, callerParticipant.ID)
		avatarURL := uc.resolveAvatarURL(ctx, room, callerParticipant.ID)

		var latestMsg *string
		if msg, err := uc.chatMessageRepo.FindLatestByRoom(ctx, room.ID); err == nil {
			latestMsg = &msg.Content
		}

		since := member.JoinedAt
		if member.LastReadAt != nil {
			since = *member.LastReadAt
		}
		unreadCount, _ := uc.chatMessageRepo.CountByRoomAfter(ctx, room.ID, since)

		summary := ChatRoomSummary{
			RoomID:      int64(room.ID),
			RoomType:    string(room.Type),
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			LatestMsg:   latestMsg,
			UnreadCount: unreadCount,
		}

		switch room.Type {
		case chatroom.Direct:
			out.Direct = append(out.Direct, summary)
		case chatroom.Group:
			out.Group = append(out.Group, summary)
		case chatroom.Channel:
			out.Channel = append(out.Channel, summary)
		case chatroom.Bot:
			out.Bot = append(out.Bot, summary)
		}
	}

	return out, nil
}

func (uc *GetUserChatRoomsUseCase) findOtherParticipant(
	ctx context.Context,
	room *chatroom.ChatRoom,
	callerParticipantID participant.ID,
) (*participant.Participant, error) {
	roomMembers, err := uc.chatMemberRepo.FindByRoom(ctx, room.ID)
	if err != nil {
		return nil, err
	}

	for _, m := range roomMembers {
		if m.IsDeleted {
			continue
		}
		if m.ParticipantID != callerParticipantID {
			return uc.participantRepo.FindByID(ctx, m.ParticipantID)
		}
	}

	return nil, errors.New("other participant not found")
}

func (uc *GetUserChatRoomsUseCase) resolveDisplayName(
	ctx context.Context,
	room *chatroom.ChatRoom,
	callerParticipantID participant.ID,
) string {
	switch room.Type {
	case chatroom.Channel, chatroom.Group:
		return room.Name

	case chatroom.Direct, chatroom.Bot:
		p, err := uc.findOtherParticipant(ctx, room, callerParticipantID)
		if err != nil {
			return room.Name
		}

		if room.Type == chatroom.Direct && p.UserID != nil {
			u, err := uc.userRepo.FindByID(ctx, *p.UserID)
			if err != nil {
				return room.Name
			}
			return u.Name
		}

		if room.Type == chatroom.Bot && p.AgentID != nil {
			a, err := uc.agentRepo.FindByID(ctx, agent.ID(*p.AgentID))
			if err != nil {
				return room.Name
			}
			return a.Name
		}

		return room.Name
	}

	return room.Name
}

func (uc *GetUserChatRoomsUseCase) resolveAvatarURL(
	ctx context.Context,
	room *chatroom.ChatRoom,
	callerParticipantID participant.ID,
) *string {
	switch room.Type {
	case chatroom.Group, chatroom.Channel:
		if room.AvatarName != "" {
			s := room.AvatarName
			return &s
		}
		return nil

	case chatroom.Direct, chatroom.Bot:
		p, err := uc.findOtherParticipant(ctx, room, callerParticipantID)
		if err != nil {
			return nil
		}

		if room.Type == chatroom.Direct && p.UserID != nil {
			u, err := uc.userRepo.FindByID(ctx, *p.UserID)
			if err != nil {
				return nil
			}
			if u.Avatar == "" {
				return nil
			}
			return &u.Avatar
		}

		return nil
	}

	return nil
}
