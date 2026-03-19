package chat

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmessage"

func toChatMessageDomain(rec *ChatMessageRecord) (*chatmessage.ChatMessage, error) {
	return &chatmessage.ChatMessage{
		ID:        rec.ID,
		RoomID:    rec.RoomID,
		SenderID:  rec.SenderID,
		Content:   rec.Content,
		Type:      rec.Type,
		ReplyToID: rec.ReplyToID,
		IsEdited:  rec.IsEdited,
		IsDeleted: rec.IsDeleted,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
	}, nil
}

func toChatMessageRecord(msg *chatmessage.ChatMessage) *ChatMessageRecord {
	return &ChatMessageRecord{
		ID:        msg.ID,
		RoomID:    msg.RoomID,
		SenderID:  msg.SenderID,
		Content:   msg.Content,
		Type:      msg.Type,
		ReplyToID: msg.ReplyToID,
		IsEdited:  msg.IsEdited,
		IsDeleted: msg.IsDeleted,
		CreatedAt: msg.CreatedAt,
		UpdatedAt: msg.UpdatedAt,
	}
}
