package chat

import (
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func toChatMessageDomain(rec *ChatMessageRecord) (*chatmessage.ChatMessage, error) {
	return &chatmessage.ChatMessage{
		ID:               rec.ID,
		RoomID:           rec.RoomID,
		SenderID:         rec.SenderID,
		SenderDeviceID:   shared.DeviceID(parseUUIDOrNil(rec.SenderDeviceID)),
		SenderKeyVersion: rec.SenderKeyVersion,
		Content:          rec.Content,
		Type:             rec.Type,
		ReplyToID:        rec.ReplyToID,
		IsEdited:         rec.IsEdited,
		IsDeleted:        rec.IsDeleted,
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
	}, nil
}

func toChatMessageRecord(msg *chatmessage.ChatMessage) *ChatMessageRecord {
	return &ChatMessageRecord{
		ID:               msg.ID,
		RoomID:           msg.RoomID,
		SenderID:         msg.SenderID,
		SenderDeviceID:   msg.SenderDeviceID.String(),
		SenderKeyVersion: msg.SenderKeyVersion,
		Content:          msg.Content,
		Type:             msg.Type,
		ReplyToID:        msg.ReplyToID,
		IsEdited:         msg.IsEdited,
		IsDeleted:        msg.IsDeleted,
		CreatedAt:        msg.CreatedAt,
		UpdatedAt:        msg.UpdatedAt,
	}
}

func parseUUIDOrNil(raw string) [16]byte {
	id, err := shared.ParseDeviceID(raw)
	if err != nil {
		return [16]byte{}
	}
	return [16]byte(id)
}
