package chat

import "github.com/HiroLiang/tentserv-chat-server/internal/domain/chatinvitation"

func toInvitationDomain(rec *ChatInvitationRecord) *chatinvitation.ChatInvitation {
	return &chatinvitation.ChatInvitation{
		ID:             rec.ID,
		RoomID:         rec.RoomID,
		InviterID:      rec.InviterID,
		InviteeID:      rec.InviteeID,
		Status:         rec.Status,
		InvitationType: rec.InvitationType,
		ExpiresAt:      rec.ExpiresAt,
		CreatedAt:      rec.CreatedAt,
		UpdatedAt:      rec.UpdatedAt,
	}
}

func toInvitationRecord(inv *chatinvitation.ChatInvitation) *ChatInvitationRecord {
	return &ChatInvitationRecord{
		ID:             inv.ID,
		RoomID:         inv.RoomID,
		InviterID:      inv.InviterID,
		InviteeID:      inv.InviteeID,
		Status:         inv.Status,
		InvitationType: inv.InvitationType,
		ExpiresAt:      inv.ExpiresAt,
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
