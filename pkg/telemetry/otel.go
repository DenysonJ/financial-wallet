package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Config holds telemetry configuration.
type Config struct {
	ServiceName string
	Enabled     bool
}

// Option configures the telemetry setup.
type Option func(*options)

type options struct {
	spanExporter   sdktrace.SpanExporter
	metricExporter sdkmetric.Exporter
}

// WithTraceExporter sets the span exporter (e.g., gRPC, HTTP, stdout).
func WithTraceExporter(e sdktrace.SpanExporter) Option {
	return func(o *options) { o.spanExporter = e }
}

// WithMetricExporter sets the metric exporter (e.g., gRPC, HTTP, stdout).
func WithMetricExporter(e sdkmetric.Exporter) Option {
	return func(o *options) { o.metricExporter = e }
}

// Provider encapsulates tracing and metrics providers.
type Provider struct {
	tp          *sdktrace.TracerProvider
	mp          *sdkmetric.MeterProvider
	httpMetrics *HTTPMetrics
}

// Setup initializes OpenTelemetry (Traces + Metrics).
// Returns a Provider that must be shut down on application exit.
func Setup(ctx context.Context, cfg Config, opts ...Option) (*Provider, error) {
	if !cfg.Enabled {
		slog.Info("OpenTelemetry disabled")
		return &Provider{}, nil
	}

	// Apply functional options
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	// If enabled but no exporters provided, return empty provider gracefully
	if o.spanExporter == nil && o.metricExporter == nil {
		slog.Warn("OpenTelemetry enabled but no exporters configured, skipping initialization")
		return &Provider{}, nil
	}

	// Resource with service information
	res, resErr := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if resErr != nil {
		return nil, resErr
	}

	var tp *sdktrace.TracerProvider
	var mp *sdkmetric.MeterProvider

	// Traces
	if o.spanExporter != nil {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(o.spanExporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Metrics
	if o.metricExporter != nil {
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(o.metricExporter)),
			sdkmetric.WithResource(res),
		)
		otel.SetMeterProvider(mp)
	}

	// HTTP Metrics
	httpMetrics, httpErr := NewHTTPMetrics(cfg.ServiceName)
	if httpErr != nil {
		return nil, httpErr
	}

	slog.Info("OpenTelemetry initialized",
		"service", cfg.ServiceName,
	)

	return &Provider{tp: tp, mp: mp, httpMetrics: httpMetrics}, nil
}

// Shutdown shuts down telemetry providers gracefully.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp != nil {
		if tpErr := p.tp.Shutdown(ctx); tpErr != nil {
			return tpErr
		}
	}
	if p.mp != nil {
		if mpErr := p.mp.Shutdown(ctx); mpErr != nil {
			return mpErr
		}
	}
	return nil
}

// HTTPMetrics returns the HTTP metrics instance.
func (p *Provider) HTTPMetrics() *HTTPMetrics {
	return p.httpMetrics
}
