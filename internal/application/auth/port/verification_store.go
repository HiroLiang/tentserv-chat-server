package port

import (
	"context"
	"time"
)

type VerificationStore interface {
	Store(ctx context.Context, token string, accountID int64, ttl time.Duration) error
	Get(ctx context.Context, token string) (int64, bool, error)
	Delete(ctx context.Context, token string) error
}
