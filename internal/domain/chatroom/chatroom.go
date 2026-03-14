package chatroom

import "time"

type ChatRoom struct {
	ID          ID
	Name        string
	Description string
	AvatarName  string
	Type        RoomType
	MaxMembers  int
	AllowAgent  bool
	IsDeleted   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
