package auth

import (
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type Session struct {
	ID        string
	AccountID shared.AccountID
	UserID    shared.UserID
	DeviceID  shared.DeviceID
	Token     TokenPair
	CreatedAt time.Time
}

type CreateSessionInput struct {
	AccountID shared.AccountID
	UserID    shared.UserID
	DeviceID  shared.DeviceID
}
