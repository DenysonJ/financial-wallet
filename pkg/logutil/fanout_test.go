package logutil

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Handle ---

func TestFanoutHandler_Handle_dispatchesToAllHandlers(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	fanout := NewFanoutHandler(h1, h2)
	logger := slog.New(fanout)

	logger.Info("test message", "key", "value")

	assert.Contains(t, buf1.String(), "test message", "primary handler should receive message")
	assert.Contains(t, buf2.String(), "test message", "secondary handler should receive message")
	assert.Contains(t, buf1.String(), `"key":"value"`, "primary handler should receive attrs")
	assert.Contains(t, buf2.String(), `"key":"value"`, "secondary handler should receive attrs")
}

func TestFanoutHandler_Handle_primaryFailureReturnsError(t *testing.T) {
	failing := &failingHandler{}
	var buf bytes.Buffer
	h2 := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	fanout := NewFanoutHandler(failing, h2)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	handleErr := fanout.Handle(context.Background(), record)

	assert.Error(t, handleErr, "primary failure should return error")
}

func TestFanoutHandler_Handle_secondaryFailureDoesNotBlock(t *testing.T) {
	var buf bytes.Buffer
	h1 := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	failing := &failingHandler{}

	fanout := NewFanoutHandler(h1, failing)
	logger := slog.New(fanout)

	logger.Info("should still work")

	assert.Contains(t, buf.String(), "should still work", "primary handler should receive message despite secondary failure")
}

func TestFanoutHandler_Handle_preservesLogLevel(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})

	fanout := NewFanoutHandler(h)
	logger := slog.New(fanout)

	logger.Warn("warn message")

	assert.Contains(t, buf.String(), `"level":"WARN"`, "log level should be preserved")
}

func TestFanoutHandler_Handle_allSecondaryFailures(t *testing.T) {
	var buf bytes.Buffer
	h1 := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	fanout := NewFanoutHandler(h1, &failingHandler{}, &failingHandler{})

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "multi-fail", 0)
	handleErr := fanout.Handle(context.Background(), record)

	assert.NoError(t, handleErr, "primary success means no error despite secondary failures")
	assert.Contains(t, buf.String(), "multi-fail", "primary handler should still receive message")
}

// --- Enabled ---

func TestFanoutHandler_Enabled_trueIfAnyEnabled(t *testing.T) {
	debugHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug})
	errorHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError})

	fanout := NewFanoutHandler(errorHandler, debugHandler)

	assert.True(t, fanout.Enabled(context.Background(), slog.LevelDebug),
		"should be enabled if any handler accepts the level")
	assert.True(t, fanout.Enabled(context.Background(), slog.LevelInfo),
		"should be enabled if any handler accepts the level")
}

func TestFanoutHandler_Enabled_falseIfNoneEnabled(t *testing.T) {
	errorOnly := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError})

	fanout := NewFanoutHandler(errorOnly)

	assert.False(t, fanout.Enabled(context.Background(), slog.LevelDebug),
		"should be disabled if no handler accepts the level")
	assert.False(t, fanout.Enabled(context.Background(), slog.LevelInfo),
		"should be disabled if no handler accepts the level")
}

func TestFanoutHandler_Enabled_primaryEnabledSuffices(t *testing.T) {
	debugHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug})
	errorHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError})

	fanout := NewFanoutHandler(debugHandler, errorHandler)

	assert.True(t, fanout.Enabled(context.Background(), slog.LevelInfo),
		"should be enabled when primary accepts the level")
}

// --- WithAttrs ---

func TestFanoutHandler_WithAttrs_propagatesToAllHandlers(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	fanout := NewFanoutHandler(h1, h2)
	withAttrs := fanout.WithAttrs([]slog.Attr{slog.String("service", "test-svc")})

	logger := slog.New(withAttrs)
	logger.Info("attrs test")

	assert.Contains(t, buf1.String(), `"service":"test-svc"`, "primary should have propagated attr")
	assert.Contains(t, buf2.String(), `"service":"test-svc"`, "secondary should have propagated attr")
}

// --- WithGroup ---

func TestFanoutHandler_WithGroup_propagatesToAllHandlers(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	h2 := slog.NewJSONHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	fanout := NewFanoutHandler(h1, h2)
	withGroup := fanout.WithGroup("shared")

	logger := slog.New(withGroup)
	logger.Info("group test", "k", "v")

	assert.Contains(t, buf1.String(), "shared", "primary should have propagated group")
	assert.Contains(t, buf2.String(), "shared", "secondary should have propagated group")
}

// --- Test Double ---

// failingHandler always returns an error from Handle.
type failingHandler struct{}

func (f *failingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (f *failingHandler) Handle(_ context.Context, _ slog.Record) error {
	return errors.New("handler failed")
}
func (f *failingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return f }
func (f *failingHandler) WithGroup(_ string) slog.Handler      { return f }
