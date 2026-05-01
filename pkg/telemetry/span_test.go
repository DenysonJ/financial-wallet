package telemetry_test

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/DenysonJ/financial-wallet/pkg/apperror"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// newRecorder returns a tracer provider backed by an in-memory span recorder.
func newRecorder(t *testing.T) (*tracetest.SpanRecorder, *sdktrace.TracerProvider) {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return sr, tp
}

func TestFailSpan_GivenError_WhenInvoked_ThenRecordsAndMarksError(t *testing.T) {
	sr, tp := newRecorder(t)
	_, span := tp.Tracer("test").Start(context.Background(), "op")

	sentinelErr := errors.New("boom")
	telemetry.FailSpan(span, sentinelErr, "operation failed")
	span.End()

	snaps := sr.Ended()
	if len(snaps) != 1 {
		t.Fatalf("expected 1 span, got %d", len(snaps))
	}
	got := snaps[0]

	if got.Status().Code != codes.Error {
		t.Errorf("expected status code Error, got %v", got.Status().Code)
	}
	if got.Status().Description != "operation failed" {
		t.Errorf("expected description 'operation failed', got %q", got.Status().Description)
	}

	events := got.Events()
	if len(events) == 0 {
		t.Fatal("expected at least one event (RecordError), got none")
	}
	hasErrorType := false
	for _, ev := range events {
		for _, a := range ev.Attributes {
			if a.Key == "error.type" && a.Value.AsString() == "*errors.errorString" {
				hasErrorType = true
			}
		}
	}
	if !hasErrorType {
		t.Error("expected event with error.type='*errors.errorString' attribute")
	}
}

func TestWarnSpan_GivenAttributes_WhenInvoked_ThenAddsThemAndKeepsStatusUnset(t *testing.T) {
	sr, tp := newRecorder(t)
	_, span := tp.Tracer("test").Start(context.Background(), "op")

	telemetry.WarnSpan(span,
		attribute.String("app.result", "not_found"),
		attribute.String("app.id", "123"),
	)
	span.End()

	got := sr.Ended()[0]

	if got.Status().Code == codes.Error {
		t.Errorf("expected non-Error status, got %v", got.Status().Code)
	}

	attrs := map[string]string{}
	for _, a := range got.Attributes() {
		attrs[string(a.Key)] = a.Value.AsString()
	}
	if attrs["app.result"] != "not_found" {
		t.Errorf("missing attribute app.result=not_found, got %+v", attrs)
	}
	if attrs["app.id"] != "123" {
		t.Errorf("missing attribute app.id=123, got %+v", attrs)
	}
}

func TestOkSpan_GivenSpan_WhenInvoked_ThenSetsOkStatus(t *testing.T) {
	sr, tp := newRecorder(t)
	_, span := tp.Tracer("test").Start(context.Background(), "op")

	telemetry.OkSpan(span)
	span.End()

	got := sr.Ended()[0]
	if got.Status().Code != codes.Ok {
		t.Errorf("expected status code Ok, got %v", got.Status().Code)
	}
}

func TestIsExpected_GivenError_WhenChecked_ThenReportsRegistration(t *testing.T) {
	before := len(apperror.DomainSentinels)
	t.Cleanup(func() { apperror.DomainSentinels = apperror.DomainSentinels[:before] })

	sentinel := errors.New("test: expected sentinel")
	apperror.Register(sentinel)

	unregistered := errors.New("random failure")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil returns false", nil, false},
		{"unregistered returns false", unregistered, false},
		{"registered sentinel returns true", sentinel, true},
		{"wrapped sentinel via errors.Join returns true", errors.Join(unregistered, sentinel), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := telemetry.IsExpected(tc.err); got != tc.want {
				t.Errorf("IsExpected(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestHelpers_GivenNilSpan_WhenInvoked_ThenAreNoop(t *testing.T) {
	// Must not panic.
	telemetry.FailSpan(nil, errors.New("x"), "msg")
	telemetry.WarnSpan(nil, attribute.String("k", "v"))
	telemetry.OkSpan(nil)
}
