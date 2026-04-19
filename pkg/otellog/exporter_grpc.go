package otellog

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
)

// ExporterConfig holds OTLP gRPC log exporter parameters. CollectorURL points
// at the OpenTelemetry Collector's OTLP gRPC endpoint (usually :4317) — the
// same endpoint used for traces and metrics.
type ExporterConfig struct {
	CollectorURL string
	Insecure     bool
}

// NewGRPCExporter creates an OTLP gRPC log exporter ready to be plugged into
// Setup via WithExporter. Mirrors the shape of pkg/telemetry/otelgrpc so all
// three signals (traces, metrics, logs) use the same wiring style.
func NewGRPCExporter(ctx context.Context, cfg ExporterConfig) (*otlploggrpc.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.CollectorURL),
	}
	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	return otlploggrpc.New(ctx, opts...)
}
