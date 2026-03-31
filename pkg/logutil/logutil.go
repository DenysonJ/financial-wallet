package logutil

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
)

// Context key type for safe context value storage.
type contextKey string

const logContextKey contextKey = "log_context"

// Step constants for identifying the layer in logs.
const (
	StepHandler    = "handler"
	StepUseCase    = "usecase"
	StepRepository = "repository"
	StepCache      = "cache"
	StepMiddleware = "middleware"
)

// Known domain error codes that should NOT include stack traces.
var domainErrorCodes = map[string]bool{
	"NOT_FOUND":       true,
	"INVALID_REQUEST": true,
	"CONFLICT":        true,
	"VALIDATION":      true,
	"DUPLICATE":       true,
	"FORBIDDEN":       true,
	"UNAUTHORIZED":    true,
}

// LogContext holds structured logging context across layers.
type LogContext struct {
	RequestID     string
	TraceID       string
	CallerService string
	Step          string
	Resource      string
	Action        string
	Extra         map[string]any
}

// Inject stores a LogContext in the context.
func Inject(ctx context.Context, lc LogContext) context.Context {
	return context.WithValue(ctx, logContextKey, lc)
}

// Extract retrieves a LogContext from the context.
func Extract(ctx context.Context) (LogContext, bool) {
	lc, ok := ctx.Value(logContextKey).(LogContext)
	return lc, ok
}

// WithStep returns a new LogContext copy with the step set.
func (lc LogContext) WithStep(step string) LogContext {
	lc.Step = step
	return lc
}

// WithResource returns a new LogContext copy with the resource set.
func (lc LogContext) WithResource(resource string) LogContext {
	lc.Resource = resource
	return lc
}

// WithAction returns a new LogContext copy with the action set.
func (lc LogContext) WithAction(action string) LogContext {
	lc.Action = action
	return lc
}

// ToSlogAttrs converts the LogContext to flat key-value pairs for slog.
// Returns []any (alternating key, value) which is ergonomic for slog API calls.
func (lc LogContext) ToSlogAttrs() []any {
	// 6 fixed fields * 2 (key+value) + extra entries * 2
	attrs := make([]any, 0, 12+len(lc.Extra)*2)

	if lc.RequestID != "" {
		attrs = append(attrs, "request_id", lc.RequestID)
	}
	if lc.TraceID != "" {
		attrs = append(attrs, "trace_id", lc.TraceID)
	}
	if lc.CallerService != "" {
		attrs = append(attrs, "caller_service", lc.CallerService)
	}
	if lc.Step != "" {
		attrs = append(attrs, "step", lc.Step)
	}
	if lc.Resource != "" {
		attrs = append(attrs, "resource", lc.Resource)
	}
	if lc.Action != "" {
		attrs = append(attrs, "action", lc.Action)
	}

	for k, v := range lc.Extra {
		attrs = append(attrs, k, v)
	}

	return attrs
}

// ErrorLogFields returns slog key-value pairs for error logging.
// Includes the error code in log attributes. Stack trace is only included
// when the code is empty or indicates an internal error (not a known domain error code).
func ErrorLogFields(err error, code string) []any {
	attrs := []any{
		"error.message", err.Error(),
		"error.code", code,
	}

	if !domainErrorCodes[code] {
		attrs = append(attrs, "error.stack", getStackTrace())
	}

	return attrs
}

// getStackTrace returns a formatted stack trace string.
func getStackTrace() string {
	const maxDepth = 10
	var pcs [maxDepth]uintptr
	n := runtime.Callers(3, pcs[:]) // skip getStackTrace, ErrorLogFields, and caller

	frames := runtime.CallersFrames(pcs[:n])
	var sb strings.Builder

	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			fmt.Fprintf(&sb, "%s:%d %s\n", frame.File, frame.Line, frame.Function)
		}
		if !more {
			break
		}
	}

	return sb.String()
}

// LogInfo logs an info message with LogContext attributes from context.
func LogInfo(ctx context.Context, msg string, extraArgs ...any) {
	args := contextArgsFromCtx(ctx)
	args = append(args, extraArgs...)
	slog.InfoContext(ctx, msg, args...)
}

// LogError logs an error message with LogContext attributes from context.
func LogError(ctx context.Context, msg string, extraArgs ...any) {
	args := contextArgsFromCtx(ctx)
	args = append(args, extraArgs...)
	slog.ErrorContext(ctx, msg, args...)
}

// LogWarn logs a warning message with LogContext attributes from context.
func LogWarn(ctx context.Context, msg string, extraArgs ...any) {
	args := contextArgsFromCtx(ctx)
	args = append(args, extraArgs...)
	slog.WarnContext(ctx, msg, args...)
}

// contextArgsFromCtx extracts LogContext from context and returns slog args.
func contextArgsFromCtx(ctx context.Context) []any {
	lc, ok := Extract(ctx)
	if !ok {
		return nil
	}
	return lc.ToSlogAttrs()
}
