package chat

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/chatroom"

func toChatRoomDomain(rec *ChatRoomRecord) (*chatroom.ChatRoom, error) {
	room := &chatroom.ChatRoom{
		ID:         rec.ID,
		Type:       rec.Type,
		MaxMembers: rec.MaxMembers,
		AllowAgent: rec.AllowAgent,
		IsDeleted:  rec.IsDeleted,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}
	if rec.Name != nil {
		room.Name = *rec.Name
	}
	if rec.Description != nil {
		room.Description = *rec.Description
	}
	if rec.AvatarName != nil {
		room.AvatarName = *rec.AvatarName
	}
	return room, nil
}

func toChatRoomRecord(r *chatroom.ChatRoom) *ChatRoomRecord {
	rec := &ChatRoomRecord{
		ID:         r.ID,
		Type:       r.Type,
		MaxMembers: r.MaxMembers,
		AllowAgent: r.AllowAgent,
		IsDeleted:  r.IsDeleted,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if r.Name != "" {
		rec.Name = &r.Name
	}
	if r.Description != "" {
		rec.Description = &r.Description
	}
	if r.AvatarName != "" {
		rec.AvatarName = &r.AvatarName
	}
	return rec
}
