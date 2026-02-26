package chatgroup

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/user"
)

type ChatGroup struct {
	ID          ID
	Name        string
	Description string
	AvatarURL   string
	Type        GroupType
	MaxMembers  int
	IsDeleted   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   *user.ID // nullable: nil for bot/system groups
}

func NewDirectGroup(creatorID user.ID) *ChatGroup {
	return &ChatGroup{
		Type:       Direct,
		MaxMembers: 2,
		CreatedBy:  &creatorID,
	}
}

func NewGroup(name, description string, maxMembers int, creatorID user.ID) *ChatGroup {
	return &ChatGroup{
		Name:        name,
		Description: description,
		Type:        Group,
		MaxMembers:  maxMembers,
		CreatedBy:   &creatorID,
	}
}

func NewChannel(name, description string, maxMembers int, creatorID user.ID) *ChatGroup {
	return &ChatGroup{
		Name:        name,
		Description: description,
		Type:        Channel,
		MaxMembers:  maxMembers,
		CreatedBy:   &creatorID,
	}
}

func NewBotGroup(name, description string) *ChatGroup {
	return &ChatGroup{
		Name:        name,
		Description: description,
		Type:        Bot,
		MaxMembers:  2,
		CreatedBy:   nil,
	}
}

func (g *ChatGroup) IsActive() bool {
	return !g.IsDeleted
}

func (g *ChatGroup) IsDirect() bool {
	return g.Type == Direct
}

func (g *ChatGroup) IsFull(currentCount int) bool {
	return currentCount >= g.MaxMembers
}
