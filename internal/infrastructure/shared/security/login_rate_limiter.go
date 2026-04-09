package security

import (
	"context"

	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
)

// RedisLoginRateLimiter enforces a per-identifier lock after consecutive failed logins.
type RedisLoginRateLimiter struct {
	redis  security.RateLimitRepository
	policy security.RateLimitPolicy
}

func NewRedisLoginRateLimiter(
	redis security.RateLimitRepository,
	policy security.RateLimitPolicy,
) *RedisLoginRateLimiter {
	return &RedisLoginRateLimiter{redis: redis, policy: policy}
}

var _ appSecurity.LoginRateLimiter = (*RedisLoginRateLimiter)(nil)

func (l *RedisLoginRateLimiter) CheckLoginAttempt(ctx context.Context, _ string, identifier string) error {
	if identifier == "" {
		return nil
	}

	count, err := l.redis.Get(ctx, loginFailureKey(identifier))
	if err != nil {
		return err
	}
	if count >= l.policy.Limit {
		return security.ErrRateLimitExceeded
	}
	return nil
}

func (l *RedisLoginRateLimiter) RecordLoginAttempt(ctx context.Context, _ string, identifier string, success bool) error {
	if identifier == "" {
		return nil
	}
	if success {
		return l.redis.Reset(ctx, loginFailureKey(identifier))
	}
	_, err := l.redis.Increment(ctx, loginFailureKey(identifier), l.policy.Window)
	return err
}

func (l *RedisLoginRateLimiter) ReleaseLock(ctx context.Context, identifier string) error {
	if identifier == "" {
		return nil
	}
	return l.redis.Reset(ctx, loginFailureKey(identifier))
}

func loginFailureKey(identifier string) string {
	return "login:identifier:" + identifier
}
