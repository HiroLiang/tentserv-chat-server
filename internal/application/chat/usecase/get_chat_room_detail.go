package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/agent"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/user"
)

type GetChatRoomDetailInput struct {
	RoomID int64
}

type ChatRoomMemberInfo struct {
	MemberID      int64
	ParticipantID int64
	DisplayName   string
	AvatarURL     *string
	Role          string
	LastReadAt    *time.Time
	JoinedAt      time.Time
}

type ChatMessageInfo struct {
	MessageID int64
	SenderID  int64
	Content   string
	Type      string
	ReplyToID *int64
	IsEdited  bool
	CreatedAt time.Time
}

type GetChatRoomDetailOutput struct {
	RoomID      int64
	RoomType    string
	Name        string
	Description *string
	AvatarURL   *string
	Members     []ChatRoomMemberInfo
	Messages    []ChatMessageInfo
}

type GetChatRoomDetailUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	chatRoomRepo    chatroom.Repository
	chatMessageRepo chatmessage.Repository
	userRepo        user.Repository
	agentRepo       agent.Repository
}

func NewGetChatRoomDetailUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatRoomRepo chatroom.Repository,
	chatMessageRepo chatmessage.Repository,
	userRepo user.Repository,
	agentRepo agent.Repository,
) *GetChatRoomDetailUseCase {
	return &GetChatRoomDetailUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMessageRepo: chatMessageRepo,
		userRepo:        userRepo,
		agentRepo:       agentRepo,
	}
}

func (uc *GetChatRoomDetailUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[GetChatRoomDetailInput],
) (GetChatRoomDetailOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetChatRoomDetailOutput{}, ErrParticipantNotFound
		}
		return GetChatRoomDetailOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return GetChatRoomDetailOutput{}, ErrNotRoomMember
	}

	room, err := uc.chatRoomRepo.FindByID(ctx, roomID)
	if err != nil {
		return GetChatRoomDetailOutput{}, ErrChatRoomNotFound
	}

	allMembers, err := uc.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return GetChatRoomDetailOutput{}, err
	}

	members := make([]ChatRoomMemberInfo, 0, len(allMembers))
	for _, m := range allMembers {
		if m.IsDeleted {
			continue
		}
		info := uc.buildMemberInfo(ctx, m)
		members = append(members, info)
	}

	msgs, err := uc.chatMessageRepo.FindByRoom(ctx, roomID, 20, 0)
	if err != nil {
		return GetChatRoomDetailOutput{}, err
	}

	messages := make([]ChatMessageInfo, 0, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg.IsDeleted {
			continue
		}
		var replyTo *int64
		if msg.ReplyToID != nil {
			v := int64(*msg.ReplyToID)
			replyTo = &v
		}
		messages = append(messages, ChatMessageInfo{
			MessageID: int64(msg.ID),
			SenderID:  int64(msg.SenderID),
			Content:   msg.Content,
			Type:      string(msg.Type),
			ReplyToID: replyTo,
			IsEdited:  msg.IsEdited,
			CreatedAt: msg.CreatedAt,
		})
	}

	var description *string
	if room.Type != chatroom.Direct && room.Description != "" {
		d := room.Description
		description = &d
	}

	avatarURL := uc.resolveRoomAvatarURL(ctx, room, callerParticipant.ID)

	return GetChatRoomDetailOutput{
		RoomID:      int64(room.ID),
		RoomType:    string(room.Type),
		Name:        room.Name,
		Description: description,
		AvatarURL:   avatarURL,
		Members:     members,
		Messages:    messages,
	}, nil
}

func (uc *GetChatRoomDetailUseCase) buildMemberInfo(ctx context.Context, m *chatmember.ChatMember) ChatRoomMemberInfo {
	info := ChatRoomMemberInfo{
		MemberID:      int64(m.ID),
		ParticipantID: int64(m.ParticipantID),
		Role:          string(m.Role),
		LastReadAt:    m.LastReadAt,
		JoinedAt:      m.JoinedAt,
	}

	p, err := uc.participantRepo.FindByID(ctx, m.ParticipantID)
	if err != nil {
		return info
	}

	if p.UserID != nil {
		u, err := uc.userRepo.FindByID(ctx, *p.UserID)
		if err != nil {
			return info
		}
		info.DisplayName = u.Name
		if u.Avatar != "" {
			info.AvatarURL = &u.Avatar
		}
		return info
	}

	if p.AgentID != nil {
		a, err := uc.agentRepo.FindByID(ctx, agent.ID(*p.AgentID))
		if err != nil {
			return info
		}
		info.DisplayName = a.Name
	}

	return info
}

func (uc *GetChatRoomDetailUseCase) resolveRoomAvatarURL(
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
		roomMembers, err := uc.chatMemberRepo.FindByRoom(ctx, room.ID)
		if err != nil {
			return nil
		}
		for _, m := range roomMembers {
			if m.IsDeleted || m.ParticipantID == callerParticipantID {
				continue
			}
			p, err := uc.participantRepo.FindByID(ctx, m.ParticipantID)
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
	}

	return nil
}
