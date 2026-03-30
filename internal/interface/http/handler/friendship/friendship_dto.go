package friendship

import "time"

type FriendResponse struct {
    FriendshipID int64     `json:"friendship_id"`
    UserID       int64     `json:"user_id"`
    Name         string    `json:"name"`
    Avatar       string    `json:"avatar"`
    Status       string    `json:"status"`
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
