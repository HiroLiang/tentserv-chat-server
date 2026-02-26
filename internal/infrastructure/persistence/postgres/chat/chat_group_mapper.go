package chat

import "github.com/HiroLiang/goat-server/internal/domain/chatgroup"

func toChatGroupDomain(rec *ChatGroupRecord) (*chatgroup.ChatGroup, error) {
	return &chatgroup.ChatGroup{
		ID:          rec.ID,
		Name:        rec.Name,
		Description: rec.Description,
		AvatarURL:   rec.AvatarURL,
		Type:        rec.Type,
		MaxMembers:  rec.MaxMembers,
		IsDeleted:   rec.IsDeleted,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
		CreatedBy:   rec.CreatedBy,
	}, nil
}

func toChatGroupRecord(g *chatgroup.ChatGroup) *ChatGroupRecord {
	return &ChatGroupRecord{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		AvatarURL:   g.AvatarURL,
		Type:        g.Type,
		MaxMembers:  g.MaxMembers,
		IsDeleted:   g.IsDeleted,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
		CreatedBy:   g.CreatedBy,
	}
}
