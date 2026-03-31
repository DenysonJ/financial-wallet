package idempotency

import "context"

// Entry represents a stored idempotency response.
type Entry struct {
	Status      string `json:"status"`      // "PROCESSING" or "COMPLETED"
	StatusCode  int    `json:"status_code"` // HTTP status code of the original response
	Body        []byte `json:"body"`        // Body of the original response (JSON)
	Fingerprint string `json:"fingerprint"` // SHA-256 of the original request body
}

// StatusProcessing indicates the operation is in progress.
const StatusProcessing = "PROCESSING"

// StatusCompleted indicates the operation has finished.
const StatusCompleted = "COMPLETED"

// Store defines the interface for idempotency key storage.
// Follows Dependency Inversion: the middleware defines the interface,
// infrastructure (Redis) implements it.
type Store interface {
	// Lock attempts to acquire a lock for the given key.
	// Returns true if acquired (first request), false if already existed (retry).
	Lock(ctx context.Context, key string, fingerprint string) (bool, error)

	// Get retrieves a stored entry by key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (*Entry, error)

	// Complete marks a key as completed with the response data.
	Complete(ctx context.Context, key string, entry *Entry) error

	// Unlock removes the key (used on 5xx errors to allow retry).
	Unlock(ctx context.Context, key string) error
}
