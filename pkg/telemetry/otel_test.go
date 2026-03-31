package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// noopSpanExporter is a stub span exporter for testing.
type noopSpanExporter struct{}

func (n *noopSpanExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n *noopSpanExporter) Shutdown(_ context.Context) error {
	return nil
}

// noopMetricExporter is a stub metric exporter for testing.
type noopMetricExporter struct{}

func (n *noopMetricExporter) Temporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (n *noopMetricExporter) Aggregation(_ sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return nil
}

func (n *noopMetricExporter) Export(_ context.Context, _ *metricdata.ResourceMetrics) error {
	return nil
}

func (n *noopMetricExporter) ForceFlush(_ context.Context) error {
	return nil
}

func (n *noopMetricExporter) Shutdown(_ context.Context) error {
	return nil
}

func TestSetup_Disabled_ReturnsEmptyProvider(t *testing.T) {
	ctx := context.Background()
	cfg := Config{ServiceName: "test-svc", Enabled: false}

	provider, setupErr := Setup(ctx, cfg)

	require.NoError(t, setupErr)
	require.NotNil(t, provider)
	assert.Nil(t, provider.tp)
	assert.Nil(t, provider.mp)
	assert.Nil(t, provider.HTTPMetrics())
}

func TestSetup_EnabledNoExporters_ReturnsEmptyProvider(t *testing.T) {
	ctx := context.Background()
	cfg := Config{ServiceName: "test-svc", Enabled: true}

	provider, setupErr := Setup(ctx, cfg)

	require.NoError(t, setupErr)
	require.NotNil(t, provider)
	assert.Nil(t, provider.tp)
	assert.Nil(t, provider.mp)
	assert.Nil(t, provider.HTTPMetrics())
}

func TestSetup_WithBothExporters(t *testing.T) {
	ctx := context.Background()
	cfg := Config{ServiceName: "test-svc", Enabled: true}

	provider, setupErr := Setup(ctx, cfg,
		WithTraceExporter(&noopSpanExporter{}),
		WithMetricExporter(&noopMetricExporter{}),
	)

	require.NoError(t, setupErr)
	require.NotNil(t, provider)
	assert.NotNil(t, provider.tp)
	assert.NotNil(t, provider.mp)
	assert.NotNil(t, provider.HTTPMetrics())

	shutdownErr := provider.Shutdown(ctx)
	assert.NoError(t, shutdownErr)
}

func TestSetup_WithTraceExporterOnly(t *testing.T) {
	ctx := context.Background()
	cfg := Config{ServiceName: "test-svc", Enabled: true}

	provider, setupErr := Setup(ctx, cfg,
		WithTraceExporter(&noopSpanExporter{}),
	)

	require.NoError(t, setupErr)
	require.NotNil(t, provider)
	assert.NotNil(t, provider.tp)
	assert.Nil(t, provider.mp)
	assert.NotNil(t, provider.HTTPMetrics())

	shutdownErr := provider.Shutdown(ctx)
	assert.NoError(t, shutdownErr)
}

func TestSetup_WithMetricExporterOnly(t *testing.T) {
	ctx := context.Background()
	cfg := Config{ServiceName: "test-svc", Enabled: true}

	provider, setupErr := Setup(ctx, cfg,
		WithMetricExporter(&noopMetricExporter{}),
	)

	require.NoError(t, setupErr)
	require.NotNil(t, provider)
	assert.Nil(t, provider.tp)
	assert.NotNil(t, provider.mp)
	assert.NotNil(t, provider.HTTPMetrics())

	shutdownErr := provider.Shutdown(ctx)
	assert.NoError(t, shutdownErr)
}

func TestProvider_Shutdown_Empty(t *testing.T) {
	provider := &Provider{}

	shutdownErr := provider.Shutdown(context.Background())
	assert.NoError(t, shutdownErr)
}

func TestProvider_HTTPMetrics_Empty(t *testing.T) {
	provider := &Provider{}

	assert.Nil(t, provider.HTTPMetrics())
}
