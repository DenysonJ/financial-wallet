package redisstore

import (
	"context"
	"time"

	"github.com/DenysonJ/financial-wallet/pkg/ratelimit"
	"github.com/redis/go-redis/v9"
)

// luaRateLimit atomically increments the counter and sets TTL on first request.
// Returns: [count, ttl_milliseconds]
var luaRateLimit = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
local ttl = redis.call("PTTL", KEYS[1])
if count == 1 or ttl < 0 then
    redis.call("PEXPIRE", KEYS[1], ARGV[1])
    ttl = tonumber(ARGV[1])
end
return {count, ttl}
`)

// RedisStore implements ratelimit.Store using Redis with an atomic Lua script.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis-backed rate limit store.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Allow checks whether a request identified by key is within the rate limit.
// Uses a Lua script for atomic INCR + conditional PEXPIRE to avoid race conditions.
func (s *RedisStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (*ratelimit.Result, error) {
	windowMs := window.Milliseconds()

	result, evalErr := luaRateLimit.Run(ctx, s.client, []string{key}, windowMs).Int64Slice()
	if evalErr != nil {
		return nil, evalErr
	}

	count := int(result[0])
	ttlMs := result[1]

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	resetAt := time.Now().Add(time.Duration(ttlMs) * time.Millisecond)

	return &ratelimit.Result{
		Allowed:   count <= limit,
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}
