package usecase

import (
	"context"
	"errors"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/friendship"
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
	chatRoomRepo    chatroom.Repository
	chatMessageRepo chatmessage.Repository
	friendshipRepo  friendship.Repository
}

func NewGetChatRoomMessagesUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatRoomRepo chatroom.Repository,
	chatMessageRepo chatmessage.Repository,
	friendshipRepo ...friendship.Repository,
) *GetChatRoomMessagesUseCase {
	var fsRepo friendship.Repository
	if len(friendshipRepo) > 0 {
		fsRepo = friendshipRepo[0]
	}
	return &GetChatRoomMessagesUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatRoomRepo:    chatRoomRepo,
		chatMessageRepo: chatMessageRepo,
		friendshipRepo:  fsRepo,
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

	room, err := uc.chatRoomRepo.FindByID(ctx, roomID)
	if err != nil || room.IsDeleted {
		return GetChatRoomMessagesOutput{}, ErrChatRoomNotFound
	}

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

	// Build member ID → participant ID map once for sender resolution (H-2).
	allMembers, err := uc.chatMemberRepo.FindByRoom(ctx, roomID)
	if err != nil {
		return GetChatRoomMessagesOutput{}, err
	}
	memberParticipantMap := make(map[chatmember.ID]participant.ID, len(allMembers))
	for _, m := range allMembers {
		memberParticipantMap[m.ID] = m.ParticipantID
	}
	blockedSenders, _, _, err := blockedMembersForCaller(
		ctx,
		uc.friendshipRepo,
		uc.participantRepo,
		input.Base.Auth.UserID,
		callerParticipant.ID,
		allMembers,
	)
	if err != nil {
		return GetChatRoomMessagesOutput{}, err
	}

	var msgs []*chatmessage.ChatMessage
	if input.Data.BeforeID == 0 {
		// Initial load: fetch latest messages, then reverse to ascending order.
		msgs, err = findByRoomExcludingSenders(ctx, uc.chatMessageRepo, roomID, blockedSenders, limit, 0)
		if err == nil {
			for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
				msgs[i], msgs[j] = msgs[j], msgs[i]
			}
		}
	} else {
		msgs, err = findByRoomBeforeExcludingSenders(ctx, uc.chatMessageRepo, roomID, chatmessage.ID(input.Data.BeforeID), blockedSenders, limit)
	}
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
			MessageID:           int64(msg.ID),
			SenderID:            int64(msg.SenderID),
			SenderDeviceID:      msg.SenderDeviceID.String(),
			SenderKeyVersion:    msg.SenderKeyVersion,
			SenderParticipantID: int64(memberParticipantMap[msg.SenderID]),
			Content:             msg.Content,
			Type:                string(msg.Type),
			ReplyToID:           replyTo,
			IsEdited:            msg.IsEdited,
			CreatedAt:           msg.CreatedAt,
		})
	}

	return GetChatRoomMessagesOutput{
		Messages: messages,
		HasMore:  uint64(len(msgs)) == limit,
	}, nil
}
