package chat

import "time"

type CreateRoomRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	MaxMembers  int    `json:"max_members"`
	AllowAgent  bool   `json:"allow_agent"`
}

type CreateRoomResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	MaxMembers int       `json:"max_members"`
	AllowAgent bool      `json:"allow_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

type JoinRoomResponse struct {
	MemberID     *int64     `json:"member_id,omitempty"`
	Role         *string    `json:"role,omitempty"`
	JoinedAt     *time.Time `json:"joined_at,omitempty"`
	InvitationID *int64     `json:"invitation_id,omitempty"`
	Status       *string    `json:"status,omitempty"`
}

type ResolveInvitationRequest struct {
	Approve bool `json:"approve"`
}

type ResolveInvitationResponse struct {
	InvitationID int64      `json:"invitation_id"`
	Status       string     `json:"status"`
	MemberID     *int64     `json:"member_id,omitempty"`
	Role         *string    `json:"role,omitempty"`
	JoinedAt     *time.Time `json:"joined_at,omitempty"`
}

type ChatRoomSummaryResponse struct {
	RoomID      int64   `json:"room_id"`
	RoomType    string  `json:"room_type"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	LatestMsg   *string `json:"latest_message"`
	UnreadCount int64   `json:"unread_count"`
}

type ChatRoomMemberInfoResponse struct {
	MemberID      int64      `json:"member_id"`
	ParticipantID int64      `json:"participant_id"`
	DisplayName   string     `json:"display_name"`
	AvatarURL     *string    `json:"avatar_url"`
	Role          string     `json:"role"`
	LastReadAt    *time.Time `json:"last_read_at"`
	JoinedAt      time.Time  `json:"joined_at"`
}

type ChatMessageInfoResponse struct {
	MessageID int64     `json:"message_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *int64    `json:"reply_to_id,omitempty"`
	IsEdited  bool      `json:"is_edited"`
	CreatedAt time.Time `json:"created_at"`
}

type GetChatRoomDetailResponse struct {
	RoomID      int64                        `json:"room_id"`
	RoomType    string                       `json:"room_type"`
	Name        string                       `json:"name"`
	Description *string                      `json:"description,omitempty"`
	AvatarURL   *string                      `json:"avatar_url"`
	Members     []ChatRoomMemberInfoResponse `json:"members"`
	Messages    []ChatMessageInfoResponse    `json:"messages"`
}

type GetChatRoomMessagesRequest struct {
	BeforeID int64  `form:"before_id"`
	Limit    uint64 `form:"limit"`
}

type GetChatRoomMessagesResponse struct {
	Messages []ChatMessageInfoResponse `json:"messages"`
	HasMore  bool                      `json:"has_more"`
}

type MemberStatusInfoResponse struct {
	MemberID   int64      `json:"member_id"`
	LastReadAt *time.Time `json:"last_read_at"`
}

type UpdateMemberStatusResponse struct {
	Members []MemberStatusInfoResponse `json:"members"`
}

type GetUserChatRoomsResponse struct {
	Direct  []ChatRoomSummaryResponse `json:"direct"`
	Group   []ChatRoomSummaryResponse `json:"group"`
	Channel []ChatRoomSummaryResponse `json:"channel"`
	Bot     []ChatRoomSummaryResponse `json:"bot"`
}

type SendMessageRequest struct {
	Content   string `json:"content"   binding:"required"`
	Type      string `json:"type"      binding:"required,oneof=text image file"`
	ReplyToID *int64 `json:"reply_to_id,omitempty"`
}

type SendMessageResponse struct {
	MessageID int64     `json:"message_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *int64    `json:"reply_to_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UploadRoomMediaResponse struct {
	Path     string `json:"path"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}
