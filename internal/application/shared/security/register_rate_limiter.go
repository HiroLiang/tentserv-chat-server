package security

import "context"

// RegisterRateLimiter limits account registration attempts by client IP.
type RegisterRateLimiter interface {
	CheckRegisterAttempt(ctx context.Context, ip string) error
}
