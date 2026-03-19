package security

import (
	"context"
	"time"

	securityApp "github.com/HiroLiang/tentserv-chat-server/internal/application/shared/security"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/security"
)

type RedisRateLimiter struct {
	redis        security.RateLimitRepository
	globalPolicy security.RateLimitPolicy
	ipPolicy     security.RateLimitPolicy
}

func NewRedisRateLimiter(
	redis security.RateLimitRepository,
	globalPolicy security.RateLimitPolicy,
	ipPolicy security.RateLimitPolicy,
) *RedisRateLimiter {
	return &RedisRateLimiter{redis, globalPolicy, ipPolicy}
}

var _ securityApp.RateLimiter = (*RedisRateLimiter)(nil)

func (limiter RedisRateLimiter) CheckGlobal(ctx context.Context) error {

	// Increment global rate limit counter with the sliding window and get count
	c, err := limiter.redis.IncrementSliding(ctx, "global", limiter.globalPolicy.Window, time.Now())
	if err != nil {
		return err
	}

	// Check if the count exceeds the limit
	if c > limiter.globalPolicy.Limit {
		return security.ErrRateLimitExceeded
	}

	return nil
}

func (limiter RedisRateLimiter) CheckIP(ctx context.Context, ip string) error {

	// get count
	c, err := limiter.redis.Get(ctx, "ip:"+ip)
	if err != nil {
		return err
	}

	// Check if the count exceeds the limit
	if c > limiter.ipPolicy.Limit {
		return security.ErrRateLimitExceeded
	}

	// Increment IP rate limit counter
	c, err = limiter.redis.Increment(ctx, "ip:"+ip, limiter.ipPolicy.Window)
	if err != nil {
		return err
	}

	return nil
}
