package security

import (
	"context"

	appSecurity "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
)

// RedisRegisterRateLimiter enforces a per-IP cap on account registration attempts.
type RedisRegisterRateLimiter struct {
	redis  security.RateLimitRepository
	policy security.RateLimitPolicy
}

func NewRedisRegisterRateLimiter(
	redis security.RateLimitRepository,
	policy security.RateLimitPolicy,
) *RedisRegisterRateLimiter {
	return &RedisRegisterRateLimiter{redis: redis, policy: policy}
}

var _ appSecurity.RegisterRateLimiter = (*RedisRegisterRateLimiter)(nil)

func (l *RedisRegisterRateLimiter) CheckRegisterAttempt(ctx context.Context, ip string) error {
	c, err := l.redis.Get(ctx, "register:ip:"+ip)
	if err != nil {
		return err
	}
	if c > l.policy.Limit {
		return security.ErrRateLimitExceeded
	}
	_, err = l.redis.Increment(ctx, "register:ip:"+ip, l.policy.Window)
	return err
}
