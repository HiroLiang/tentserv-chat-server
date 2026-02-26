package chat

// LastMessagePreviewResponse is a preview of the most recent message in a group.
type LastMessagePreviewResponse struct {
	Content    string `json:"content"`
	SenderName string `json:"senderName"`
	Timestamp  string `json:"timestamp"`
}

// ChatGroupResponse represents a single chat group entry for the group list.
type ChatGroupResponse struct {
	ID          int64                       `json:"id"`
	Type        string                      `json:"type"`
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	AvatarURL   string                      `json:"avatarUrl,omitempty"`
	LastMessage *LastMessagePreviewResponse `json:"lastMessage,omitempty"`
	UnreadCount int64                       `json:"unreadCount"`
	MemberCount int                         `json:"memberCount"`
}

// GetMyGroupsResponse is the response body for GET /api/chat/groups.
type GetMyGroupsResponse struct {
	Groups []ChatGroupResponse `json:"groups"`
}

// ChatMessageResponse represents a single chat message.
type ChatMessageResponse struct {
	ID           int64  `json:"id"`
	ChatID       int64  `json:"chatId"`
	SenderID     int64  `json:"senderId"`
	SenderName   string `json:"senderName"`
	SenderAvatar string `json:"senderAvatar,omitempty"`
	Content      string `json:"content"`
	Type         string `json:"type"`
	ReplyToID    *int64 `json:"replyToId,omitempty"`
	IsEdited     bool   `json:"isEdited"`
	IsMe         bool   `json:"isMe"`
	Timestamp    string `json:"timestamp"`
}

// GetGroupMessagesResponse is the response body for GET /api/chat/groups/:id/messages.
type GetGroupMessagesResponse struct {
	Messages   []ChatMessageResponse `json:"messages"`
	NextCursor *int64                `json:"nextCursor,omitempty"`
	HasMore    bool                  `json:"hasMore"`
}

// CreateGroupRequest is the request body for POST /api/chat/groups.
type CreateGroupRequest struct {
	Type        string  `json:"type" binding:"required"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	MemberIDs   []int64 `json:"memberIds"`
	MaxMembers  *int    `json:"maxMembers,omitempty"`
}

// CreateGroupResponse is the response body for POST /api/chat/groups.
type CreateGroupResponse struct {
	Group     ChatGroupResponse `json:"group"`
	IsCreated bool              `json:"isCreated"`
}
