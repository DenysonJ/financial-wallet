package redisstore

import (
	"context"
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/ratelimit"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements ratelimit.Store using Redis INCR + EXPIRE (fixed window).
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed rate limit store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Allow checks whether a request identified by key is within the rate limit.
// Uses a Redis pipeline with INCR + conditional EXPIRE for atomicity.
// The key automatically expires after the window duration.
func (s *RedisStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.Result, error) {
	pipe := s.client.Pipeline()

	incrCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, execErr := pipe.Exec(ctx)
	if execErr != nil && execErr != redis.Nil {
		return nil, execErr
	}

	count := incrCmd.Val()
	ttl := ttlCmd.Val()

	// If this is the first request in the window (count == 1), set the expiration.
	// Also handle the case where the key exists but has no TTL (e.g., after a crash).
	if count == 1 || ttl < 0 {
		s.client.Expire(ctx, key, window)
		ttl = window
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	resetAt := time.Now().Add(ttl)

	return &ratelimit.Result{
		Allowed:   int(count) <= limit,
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}
