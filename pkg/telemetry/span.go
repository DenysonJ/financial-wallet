package telemetry

import (
	"context"
	"errors"
	"fmt"

	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/DenysonJ/financial-wallet/pkg/apperror"
)

// FailSpan records an UNEXPECTED error on the span and marks its status as
// Error. Use only for errors that should trigger alerts — timeouts, dropped
// connections, 5xx from dependencies, panics, or any error not classified as
// an expected domain sentinel.
//
// For expected business errors (not-found, validation, conflict) use
// WarnSpan instead so the span stays Ok and dashboards do not treat the
// outcome as a failure.
func FailSpan(span trace.Span, err error, msg string) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err,
		trace.WithAttributes(attribute.String("error.type", fmt.Sprintf("%T", err))),
		trace.WithStackTrace(true),
	)
	span.SetStatus(codes.Error, msg)
}

// WarnSpan adds semantic attributes to the span without marking it as Error.
// Use for expected outcomes (domain sentinels, validation errors, 4xx) that
// should be visible in traces but must not pollute error-rate dashboards.
func WarnSpan(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil || len(attrs) == 0 {
		return
	}
	span.SetAttributes(attrs...)
}

// OkSpan marks the span status as Ok explicitly. Call it on the happy path
// right before returning so the successful outcome is explicit in the trace.
func OkSpan(span trace.Span) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, "")
}

// IsExpected reports whether err is (or wraps) a registered domain sentinel.
//
// The canonical list lives in apperror.DomainSentinels and is populated at
// init() time by the handler package (single source of truth shared with the
// HTTP error translation). A true result means the error is part of normal
// business flow; callers should use WarnSpan. A false result means the error
// is unexpected; callers should use FailSpan.
func IsExpected(err error) bool {
	if err == nil {
		return false
	}
	for _, sentinel := range apperror.Sentinels() {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}

// ResultAttrKey is the canonical attribute key for semantic outcome on spans
const ResultAttrKey = "app.result"

// ClassifyError records err on span+logs splitting expected vs unexpected.
//
// resultAttr is the semantic result emitted both as the span attribute
// app.result and as the structured log field "result" (e.g. "not_found",
// "domain_error", "invalid_id"). baseMsg is the log message.
//
// Expected errors (registered domain sentinels) produce WarnSpan + LogWarn and
// the span stays Ok. Unexpected errors (everything else) produce FailSpan and
// LogError, marking the span as Error for alerting.
func ClassifyError(ctx context.Context, span trace.Span, err error, resultAttr, baseMsg string) {
	ClassifyErrorWithKey(ctx, span, err, ResultAttrKey, resultAttr, baseMsg)
}

// ClassifyErrorWithKey is the explicit form of ClassifyError that lets the
// caller override the span/log attribute key (default: "app.result")
func ClassifyErrorWithKey(ctx context.Context, span trace.Span, err error, key, resultAttr, baseMsg string) {
	if key == "" {
		key = ResultAttrKey
	}
	if IsExpected(err) {
		WarnSpan(span, attribute.String(key, resultAttr))
		logutil.LogWarn(ctx, baseMsg, "result", resultAttr, "error", err.Error())
		return
	}
	FailSpan(span, err, baseMsg)
	logutil.LogError(ctx, baseMsg+": unexpected", "error", err.Error())
}
