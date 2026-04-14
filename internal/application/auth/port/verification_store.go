package port

import (
	"context"
	"time"
)

type VerificationSession struct {
	AccountID         int64  `json:"account_id"`
	Code              string `json:"code"`
	ExpiresAtMS       int64  `json:"expires_at_ms"`
	RemainingAttempts int    `json:"remaining_attempts"`
	DeviceID          string `json:"device_id,omitempty"`
	DeviceName        string `json:"device_name,omitempty"`
	IP                string `json:"ip,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
}

type VerificationStore interface {
	Store(ctx context.Context, token string, session VerificationSession, ttl time.Duration) error
	Get(ctx context.Context, token string) (VerificationSession, bool, error)
	FindTokenByAccountID(ctx context.Context, accountID int64) (string, bool, error)
	Delete(ctx context.Context, token string) error
}
