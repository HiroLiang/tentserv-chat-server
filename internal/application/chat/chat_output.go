package chat

import "github.com/HiroLiang/goat-server/internal/domain/chatmessage"

type LastMessagePreview struct {
	Content    string
	SenderName string
	Timestamp  string
}

type ChatGroupItem struct {
	ID          int64
	Type        string
	Name        string
	Description string
	AvatarURL   string
	LastMessage *LastMessagePreview
	UnreadCount int64
	MemberCount int
}

type GetMyGroupsOutput struct {
	Groups []ChatGroupItem
}

type ChatMessageItem struct {
	ID           int64
	ChatID       int64
	SenderID     int64
	SenderName   string
	SenderAvatar string
	Content      string
	Type         chatmessage.MessageType
	ReplyToID    *int64
	IsEdited     bool
	IsMe         bool
	Timestamp    string
}

type GetGroupMessagesOutput struct {
	Messages   []ChatMessageItem
	NextCursor *int64
	HasMore    bool
}

type CreateGroupOutput struct {
	Group     ChatGroupItem
	IsCreated bool // true = newly created, false = returned existing group
}
