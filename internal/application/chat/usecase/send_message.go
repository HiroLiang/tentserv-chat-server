package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
)

type SendMessageInput struct {
	RoomID    int64
	Content   string
	Type      string
	ReplyToID *int64
}

type SendMessageOutput struct {
	MessageID int64
	RoomID    int64
	SenderID  int64
	Content   string
	Type      string
	ReplyToID *int64
	CreatedAt time.Time
}

type SendMessageUseCase struct {
	participantRepo participant.Repository
	chatMemberRepo  chatmember.Repository
	chatMessageRepo chatmessage.Repository
	broadcaster     port.Broadcaster
}

func NewSendMessageUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	chatMessageRepo chatmessage.Repository,
	broadcaster port.Broadcaster,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		participantRepo: participantRepo,
		chatMemberRepo:  chatMemberRepo,
		chatMessageRepo: chatMessageRepo,
		broadcaster:     broadcaster,
	}
}

func (uc *SendMessageUseCase) Execute(
	ctx context.Context,
	input shared.UseCaseInput[SendMessageInput],
) (SendMessageOutput, error) {
	callerParticipant, err := uc.participantRepo.FindByUserID(ctx, input.Base.Auth.UserID)
	if err != nil {
		if errors.Is(err, participant.ErrNotFound) {
			return SendMessageOutput{}, ErrParticipantNotFound
		}
		return SendMessageOutput{}, err
	}

	roomID := chatroom.ID(input.Data.RoomID)

	callerMember, err := uc.chatMemberRepo.FindByRoomAndParticipant(ctx, roomID, callerParticipant.ID)
	if err != nil || callerMember.IsDeleted {
		return SendMessageOutput{}, ErrNotRoomMember
	}

	msgType := chatmessage.MessageType(input.Data.Type)
	switch msgType {
	case chatmessage.Text, chatmessage.Image, chatmessage.File:
		// valid
	default:
		return SendMessageOutput{}, ErrInvalidMessageType
	}

	// Defense-in-depth: for file/image, reject path traversal attempts
	if msgType == chatmessage.Image || msgType == chatmessage.File {
		if strings.Contains(input.Data.Content, "..") || strings.HasPrefix(input.Data.Content, "/") {
			return SendMessageOutput{}, ErrInvalidMessageType
		}
	}

	var replyToID *chatmessage.ID
	if input.Data.ReplyToID != nil {
		id := chatmessage.ID(*input.Data.ReplyToID)
		replyToID = &id
	}

	msg := &chatmessage.ChatMessage{
		RoomID:    roomID,
		SenderID:  callerMember.ID,
		Content:   input.Data.Content,
		Type:      msgType,
		ReplyToID: replyToID,
	}

	if err := uc.chatMessageRepo.Create(ctx, msg); err != nil {
		return SendMessageOutput{}, ErrSendMessage
	}

	out := SendMessageOutput{
		MessageID: int64(msg.ID),
		RoomID:    int64(msg.RoomID),
		SenderID:  int64(msg.SenderID),
		Content:   msg.Content,
		Type:      string(msg.Type),
		CreatedAt: msg.CreatedAt,
	}
	if msg.ReplyToID != nil {
		v := int64(*msg.ReplyToID)
		out.ReplyToID = &v
	}

	go uc.fanOut(out)

	return out, nil
}

type wsMessagePayload struct {
	MessageID int64     `json:"message_id"`
	RoomID    int64     `json:"room_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *int64    `json:"reply_to_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type wsEnvelope struct {
	Type    string           `json:"type"`
	Payload wsMessagePayload `json:"payload"`
}

func (uc *SendMessageUseCase) fanOut(out SendMessageOutput) {
	ctx := context.Background()

	members, err := uc.chatMemberRepo.FindByRoom(ctx, chatroom.ID(out.RoomID))
	if err != nil {
		return
	}

	payload, err := json.Marshal(wsEnvelope{
		Type: "chat.message",
		Payload: wsMessagePayload{
			MessageID: out.MessageID,
			RoomID:    out.RoomID,
			SenderID:  out.SenderID,
			Content:   out.Content,
			Type:      out.Type,
			ReplyToID: out.ReplyToID,
			CreatedAt: out.CreatedAt,
		},
	})
	if err != nil {
		return
	}

	for _, m := range members {
		if m.IsDeleted {
			continue
		}
		p, err := uc.participantRepo.FindByID(ctx, m.ParticipantID)
		if err != nil || p.UserID == nil {
			continue
		}
		userIDStr := strconv.FormatInt(int64(*p.UserID), 10)
		uc.broadcaster.SendToUser(userIDStr, payload)
	}
}
