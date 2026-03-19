package chat

import (
	"context"
	"encoding/json"
	"strconv"

	chatUseCase "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/usecase"
	appShared "github.com/HiroLiang/tentserv-chat-server/internal/application/shared"
	domainShared "github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

// SendPayload is the payload for a "chat.send" message.
type SendPayload struct {
	RoomID    string `json:"room_id"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	ReplyToID *int64 `json:"reply_to_id,omitempty"`
}

// MessageHandler handles "chat.send" messages.
type MessageHandler struct {
	uc *chatUseCase.SendMessageUseCase
}

func NewMessageHandler(uc *chatUseCase.SendMessageUseCase) *MessageHandler {
	return &MessageHandler{uc: uc}
}

func (h *MessageHandler) Handle(client *ws.Client, payload json.RawMessage) error {
	var p SendPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	roomID, err := strconv.ParseInt(p.RoomID, 10, 64)
	if err != nil {
		logger.Log.Warn("chat.send: invalid room_id", zap.String("room_id", p.RoomID))
		return err
	}

	userID, err := domainShared.ParseUserID(client.UserID)
	if err != nil {
		logger.Log.Warn("chat.send: invalid user_id", zap.String("user_id", client.UserID))
		return err
	}

	msgType := p.Type
	if msgType == "" {
		msgType = "text"
	}

	input := appShared.UseCaseInput[chatUseCase.SendMessageInput]{
		Base: appShared.BaseContext{
			Auth: &appShared.AuthContext{
				UserID: userID,
			},
		},
		Data: chatUseCase.SendMessageInput{
			RoomID:    roomID,
			Content:   p.Content,
			Type:      msgType,
			ReplyToID: p.ReplyToID,
		},
	}

	_, err = h.uc.Execute(context.Background(), input)
	if err != nil {
		logger.Log.Error("chat.send: execute failed", zap.Error(err))
		return err
	}

	return nil
}
