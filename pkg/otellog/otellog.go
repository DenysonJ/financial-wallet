// Package otellog wires OpenTelemetry logs into slog.
//
// It plays the same role that an internal platform kit would play: it owns the
// SDK LoggerProvider lifecycle and exposes a slog.Handler (via the otelslog
// bridge) that forwards every structured log record to the global OTel
// LoggerProvider. The provider in turn exports records via OTLP — traces,
// metrics and logs ride the same collector, and each record is automatically
// tagged with the active trace_id/span_id from context.
//
// Typical wiring (in cmd/api/server.go):
//
//	exp, _ := otellog.NewGRPCExporter(ctx, otellog.ExporterConfig{
//	    CollectorURL: cfg.Otel.CollectorURL,
//	    Insecure:     cfg.Otel.Insecure,
//	})
//	lp, _ := otellog.Setup(ctx, otellog.Config{
//	    ServiceName: cfg.Otel.ServiceName,
//	    Enabled:     true,
//	}, otellog.WithExporter(exp))
//	defer lp.Shutdown(context.Background())
//
//	handler := otellog.Handler(cfg.Otel.ServiceName) // slog.Handler
//	slog.SetDefault(slog.New(handler))               // or fanout with stdout
package otellog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Config holds the setup parameters.
type Config struct {
	// ServiceName is attached as a resource attribute to every emitted record
	// and is also used as the default scope name for the slog bridge.
	ServiceName string
	// Enabled short-circuits Setup when false: the returned Provider is a
	// no-op and no global state is mutated. Handler() will then produce a
	// bridge pointing at the no-op global provider, which silently drops
	// records.
	Enabled bool
}

// Option configures Setup.
type Option func(*options)

type options struct {
	exporter sdklog.Exporter
}

// WithExporter attaches an OTLP (or any other sdklog.Exporter) exporter to
// the LoggerProvider. Without an exporter, Setup returns a no-op Provider.
func WithExporter(e sdklog.Exporter) Option {
	return func(o *options) { o.exporter = e }
}

// Provider owns the sdklog.LoggerProvider and is responsible for flushing and
// shutting it down on application exit. A zero-value Provider is safe to call
// Shutdown on (no-op).
type Provider struct {
	lp *sdklog.LoggerProvider
}

// Setup builds a LoggerProvider with the given exporter and registers it as
// the OTel global. After Setup returns successfully, Handler() produces a
// slog.Handler that forwards to this provider.
//
// If cfg.Enabled is false, or no exporter was supplied, Setup returns a no-op
// Provider without touching global state — callers don't need to branch on
// the result.
func Setup(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{}, nil
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.exporter == nil {
		return &Provider{}, nil
	}

	res, resErr := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(cfg.ServiceName),
	))
	if resErr != nil {
		return nil, resErr
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(o.exporter)),
	)
	global.SetLoggerProvider(lp)

	return &Provider{lp: lp}, nil
}

// Handler returns a slog.Handler that bridges every record into the global
// OTel LoggerProvider under the given instrumentation scope. The bridge reads
// trace_id/span_id from ctx automatically, producing correlated logs in the
// backend (e.g. Kibana's "Log rate for trace" view).
//
// Safe to call before Setup — if the global provider is still the no-op one,
// the handler becomes a silent sink. Pair it with a stdout handler via a
// fanout so local visibility never depends on OTel.
func Handler(scopeName string) slog.Handler {
	return otelslog.NewHandler(scopeName)
}

// Shutdown flushes pending batches and releases the provider. Safe to call on
// a zero-value Provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.lp == nil {
		return nil
	}
	return p.lp.Shutdown(ctx)
}
