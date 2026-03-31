package logutil

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// FanoutHandler dispatches log records to multiple slog.Handler instances.
// The primary handler's errors are returned. Secondary handler errors
// are logged to stderr without blocking.
type FanoutHandler struct {
	primary     slog.Handler
	secondaries []slog.Handler
}

// NewFanoutHandler creates a FanoutHandler with a required primary handler
// and zero or more secondary (best-effort) handlers.
func NewFanoutHandler(primary slog.Handler, secondaries ...slog.Handler) *FanoutHandler {
	return &FanoutHandler{
		primary:     primary,
		secondaries: secondaries,
	}
}

// Enabled returns true if the primary handler OR any secondary handler
// is enabled for the given level.
func (h *FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.primary.Enabled(ctx, level) {
		return true
	}
	for _, sec := range h.secondaries {
		if sec.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle dispatches the record to the primary handler and all secondaries.
// The record is cloned for each handler to prevent lazy evaluation races.
// Primary handler failure returns the error immediately.
// Secondary handler failures are logged to stderr without blocking.
func (h *FanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	if primaryErr := h.primary.Handle(ctx, record.Clone()); primaryErr != nil {
		return primaryErr
	}

	for i, sec := range h.secondaries {
		if secErr := sec.Handle(ctx, record.Clone()); secErr != nil {
			fmt.Fprintf(os.Stderr, "FanoutHandler: secondary[%d] failed: %v\n", i, secErr)
		}
	}
	return nil
}

// WithAttrs returns a new FanoutHandler with the given attributes added
// to the primary and all secondary handlers.
func (h *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	secs := make([]slog.Handler, len(h.secondaries))
	for i, sec := range h.secondaries {
		secs[i] = sec.WithAttrs(attrs)
	}
	return NewFanoutHandler(h.primary.WithAttrs(attrs), secs...)
}

// WithGroup returns a new FanoutHandler with the given group name added
// to the primary and all secondary handlers.
func (h *FanoutHandler) WithGroup(name string) slog.Handler {
	secs := make([]slog.Handler, len(h.secondaries))
	for i, sec := range h.secondaries {
		secs[i] = sec.WithGroup(name)
	}
	return NewFanoutHandler(h.primary.WithGroup(name), secs...)
}
