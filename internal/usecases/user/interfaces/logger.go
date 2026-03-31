package interfaces

import "context"

// Logger defines the contract for structured logging in use cases.
// This interface enables Dependency Inversion: use cases define what they need,
// infrastructure implements it.
type Logger interface {
	Info(ctx context.Context, msg string, attrs ...any)
	Warn(ctx context.Context, msg string, attrs ...any)
	Error(ctx context.Context, msg string, attrs ...any)
}
