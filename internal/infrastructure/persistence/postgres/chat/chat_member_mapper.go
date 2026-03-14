package chat

import "github.com/HiroLiang/goat-server/internal/domain/chatmember"

func toChatMemberDomain(rec *ChatMemberRecord) (*chatmember.ChatMember, error) {
	return &chatmember.ChatMember{
		ID:            rec.ID,
		RoomID:        rec.RoomID,
		ParticipantID: rec.ParticipantID,
		Role:          rec.Role,
		IsMuted:       rec.IsMuted,
		IsDeleted:     rec.IsDeleted,
		LastReadAt:    rec.LastReadAt,
		JoinedAt:      rec.JoinedAt,
		UpdatedAt:     rec.UpdatedAt,
		DeletedAt:     rec.DeletedAt,
	}, nil
}

func toChatMemberRecord(m *chatmember.ChatMember) *ChatMemberRecord {
	return &ChatMemberRecord{
		ID:            m.ID,
		RoomID:        m.RoomID,
		ParticipantID: m.ParticipantID,
		Role:          m.Role,
		IsMuted:       m.IsMuted,
		IsDeleted:     m.IsDeleted,
		LastReadAt:    m.LastReadAt,
		JoinedAt:      m.JoinedAt,
		UpdatedAt:     m.UpdatedAt,
		DeletedAt:     m.DeletedAt,
	}
}
