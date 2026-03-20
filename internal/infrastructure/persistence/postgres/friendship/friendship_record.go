package friendship

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type FriendshipRecord struct {
	ID        int64         `db:"id"`
	UserID    shared.UserID `db:"user_id"`
	FriendID  shared.UserID `db:"friend_id"`
	Status    string        `db:"status"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
}
