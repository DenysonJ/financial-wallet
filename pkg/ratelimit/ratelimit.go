package ratelimit

import (
	"context"
	"time"
)

// Result contains the outcome of a rate limit check.
type Result struct {
	// Allowed indicates whether the request is within the rate limit.
	Allowed bool

	// Limit is the maximum number of requests allowed in the window.
	Limit int

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// ResetAt is the time when the current window expires.
	ResetAt time.Time
}

// Store defines the interface for rate limit storage.
// Implementations should be atomic to handle concurrent requests safely.
type Store interface {
	// Allow checks whether a request identified by key is within the rate limit.
	// It atomically increments the counter and returns the result.
	// limit is the max number of requests allowed in the given window duration.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (*Result, error)
}
