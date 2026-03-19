package usecase

import (
	"context"
	"errors"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type GetChatRoomMessagesInput struct {
	RoomID   int64
	BeforeID int64
	Limit    uint64
}

type GetChatRoomMessagesOutput struct {
	Messages []ChatMessageInfo
	HasMore  bool
}

type GetChatRoomMessagesUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	chatMessageRepo chatmessage.Repository
}

func NewGetChatRoomMessagesUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatMessageRepo chatmessage.Repository,
) *GetChatRoomMessagesUseCase {
	return &GetChatRoomMessagesUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatMessageRepo: chatMessageRepo,
	}
}

func (uc *GetChatRoomMessagesUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[GetChatRoomMessagesInput],
) (GetChatRoomMessagesOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return GetChatRoomMessagesOutput{}, ErrParticipantNotFound
		}
		return GetChatRoomMessagesOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return GetChatRoomMessagesOutput{}, ErrNotRoomMember
	}

	limit := input.Data.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	msgs, err := uc.chatMessageRepo.FindByRoomBefore(ctx, roomID, chatmessage.ID(input.Data.BeforeID), limit)
	if err != nil {
		return GetChatRoomMessagesOutput{}, err
	}

	messages := make([]ChatMessageInfo, 0, len(msgs))
	for _, msg := range msgs {
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

	return GetChatRoomMessagesOutput{
		Messages: messages,
		HasMore:  uint64(len(msgs)) == limit,
	}, nil
}
