package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestDefaultApdexThresholds(t *testing.T) {
	thresholds := DefaultApdexThresholds()

	assert.Equal(t, 500*time.Millisecond, thresholds.Satisfied)
	assert.Equal(t, 2*time.Second, thresholds.Tolerating)
}

func TestNewHTTPMetrics_Success(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(nil) })

	metrics, createErr := NewHTTPMetrics("test-service")

	require.NoError(t, createErr)
	require.NotNil(t, metrics)
}

func TestNewHTTPMetrics_AllFieldsPopulated(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(nil) })

	metrics, createErr := NewHTTPMetrics("test-service")

	require.NoError(t, createErr)
	require.NotNil(t, metrics)
	assert.NotNil(t, metrics.RequestCount)
	assert.NotNil(t, metrics.RequestDuration)
	assert.NotNil(t, metrics.SlowRequests)
	assert.NotNil(t, metrics.ApdexSatisfied)
	assert.NotNil(t, metrics.ApdexTolerating)
	assert.NotNil(t, metrics.ApdexFrustrated)
}

func TestNewHTTPMetrics_ThresholdsSet(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(nil) })

	metrics, createErr := NewHTTPMetrics("test-service")

	require.NoError(t, createErr)
	require.NotNil(t, metrics)

	expected := DefaultApdexThresholds()
	assert.Equal(t, expected.Satisfied, metrics.Thresholds.Satisfied)
	assert.Equal(t, expected.Tolerating, metrics.Thresholds.Tolerating)
}

func TestHTTPMetrics_RecordRequest(t *testing.T) {
	otel.SetMeterProvider(noop.NewMeterProvider())
	t.Cleanup(func() { otel.SetMeterProvider(nil) })

	metrics, createErr := NewHTTPMetrics("test-service")
	require.NoError(t, createErr)

	tests := []struct {
		name       string
		method     string
		route      string
		statusCode int
		duration   time.Duration
	}{
		{name: "satisfied (< 500ms)", method: "GET", route: "/users", statusCode: 200, duration: 100 * time.Millisecond},
		{name: "tolerating (500ms-2s)", method: "POST", route: "/users", statusCode: 201, duration: 1 * time.Second},
		{name: "frustrated (> 2s)", method: "GET", route: "/users/:id", statusCode: 200, duration: 3 * time.Second},
		{name: "error 500", method: "GET", route: "/users", statusCode: 500, duration: 50 * time.Millisecond},
		{name: "error 404", method: "GET", route: "/users/:id", statusCode: 404, duration: 10 * time.Millisecond},
		{name: "exatamente no limite satisfied", method: "GET", route: "/test", statusCode: 200, duration: 500 * time.Millisecond},
		{name: "exatamente no limite tolerating", method: "GET", route: "/test", statusCode: 200, duration: 2 * time.Second},
		{name: "1ms acima do tolerating", method: "GET", route: "/test", statusCode: 200, duration: 2*time.Second + 1*time.Millisecond},
		{name: "zero duration", method: "GET", route: "/test", statusCode: 200, duration: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				metrics.RecordRequest(context.Background(), tt.method, tt.route, tt.statusCode, tt.duration)
			})
		})
	}
}

func TestHTTPMetrics_RecordRequest_NilReceiver(t *testing.T) {
	var metrics *HTTPMetrics
	assert.NotPanics(t, func() {
		metrics.RecordRequest(context.Background(), "GET", "/test", 200, 100*time.Millisecond)
	})
}
