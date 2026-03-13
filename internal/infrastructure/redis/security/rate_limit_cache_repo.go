package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/HiroLiang/goat-server/internal/domain/security"
	"github.com/redis/go-redis/v9"
)

type RedisRateLimitRepository struct {
	client *redis.Client
	prefix string
}

func NewRedisRateLimitRepository(client *redis.Client) *RedisRateLimitRepository {
	return &RedisRateLimitRepository{
		client: client,
		prefix: "rate_limit:",
	}
}

var _ security.RateLimitRepository = (*RedisRateLimitRepository)(nil)

func (r *RedisRateLimitRepository) Increment(ctx context.Context, key string, window time.Duration) (count int64, err error) {
	fullKey := r.prefix + key

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

func (r *RedisRateLimitRepository) Get(ctx context.Context, key string) (count int64, err error) {
	fullKey := r.prefix + key

	val, err := r.client.Get(ctx, fullKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, err
}

func (r *RedisRateLimitRepository) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefix+key).Err()
}

func (r *RedisRateLimitRepository) IncrementSliding(ctx context.Context, key string, window time.Duration, now time.Time) (count int64, err error) {
	fullKey := r.prefix + key

	windowStart := now.Add(-window).UnixMicro()
	nowMicro := now.UnixMicro()

	pipe := r.client.Pipeline()

	// Remove expired records
	pipe.ZRemRangeByScore(ctx, fullKey, "0", fmt.Sprintf("%d", windowStart))

	// Add current request
	pipe.ZAdd(ctx, fullKey, redis.Z{Score: float64(nowMicro), Member: nowMicro})

	// Count in the current window
	var c = pipe.ZCard(ctx, fullKey)

	// Set expiration time
	pipe.Expire(ctx, fullKey, window)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return c.Val(), nil
}
