package friendship

import "time"

type FriendResponse struct {
	FriendshipID int64     `json:"friendship_id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"`
	Status       string    `json:"status"`
	BlockedBy    *string   `json:"blocked_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type FriendRequestResponse struct {
	FriendshipID int64     `json:"friendship_id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
}

type ApplyFriendRequest struct {
	FriendID int64 `json:"friend_id" binding:"required"`
}

type RemovedDirectRoomResponse struct {
	RoomID    int64   `json:"room_id"`
	MemberIDs []int64 `json:"member_ids"`
}

type RemoveFriendResponse struct {
	DeletedDirectRoom *RemovedDirectRoomResponse `json:"deleted_direct_room,omitempty"`
}
