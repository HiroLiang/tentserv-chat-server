package chat

import "github.com/HiroLiang/goat-server/internal/domain/chatinvitation"

func toInvitationDomain(rec *ChatInvitationRecord) *chatinvitation.ChatInvitation {
	return &chatinvitation.ChatInvitation{
		ID:        rec.ID,
		RoomID:    rec.RoomID,
		InviterID: rec.InviterID,
		InviteeID: rec.InviteeID,
		Status:    rec.Status,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
	}
}

func toInvitationRecord(inv *chatinvitation.ChatInvitation) *ChatInvitationRecord {
	return &ChatInvitationRecord{
		ID:        inv.ID,
		RoomID:    inv.RoomID,
		InviterID: inv.InviterID,
		InviteeID: inv.InviteeID,
		Status:    inv.Status,
		CreatedAt: inv.CreatedAt,
		UpdatedAt: inv.UpdatedAt,
	}
}
