package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ApdexThresholds defines the thresholds for Apdex calculation.
type ApdexThresholds struct {
	Satisfied  time.Duration // Requests below this are "satisfied"
	Tolerating time.Duration // Requests below this are "tolerating", above is "frustrated"
}

// DefaultApdexThresholds returns sensible defaults for a REST API.
func DefaultApdexThresholds() ApdexThresholds {
	return ApdexThresholds{
		Satisfied:  500 * time.Millisecond,
		Tolerating: 2 * time.Second,
	}
}

// HTTPMetrics holds HTTP-level metrics instruments.
type HTTPMetrics struct {
	RequestCount    metric.Int64Counter
	RequestDuration metric.Float64Histogram
	SlowRequests    metric.Int64Counter
	ApdexSatisfied  metric.Int64Counter
	ApdexTolerating metric.Int64Counter
	ApdexFrustrated metric.Int64Counter
	Thresholds      ApdexThresholds
}

// NewHTTPMetrics creates HTTP metrics instruments.
func NewHTTPMetrics(serviceName string) (*HTTPMetrics, error) {
	meter := otel.Meter(serviceName)

	requestCount, countErr := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if countErr != nil {
		return nil, countErr
	}

	requestDuration, durErr := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if durErr != nil {
		return nil, durErr
	}

	slowRequests, slowErr := meter.Int64Counter(
		"http_slow_requests_total",
		metric.WithDescription("Total number of slow HTTP requests (> tolerating threshold)"),
		metric.WithUnit("{request}"),
	)
	if slowErr != nil {
		return nil, slowErr
	}

	apdexSatisfied, satErr := meter.Int64Counter(
		"http_apdex_satisfied_total",
		metric.WithDescription("Apdex: satisfied requests"),
		metric.WithUnit("{request}"),
	)
	if satErr != nil {
		return nil, satErr
	}

	apdexTolerating, tolErr := meter.Int64Counter(
		"http_apdex_tolerating_total",
		metric.WithDescription("Apdex: tolerating requests"),
		metric.WithUnit("{request}"),
	)
	if tolErr != nil {
		return nil, tolErr
	}

	apdexFrustrated, fruErr := meter.Int64Counter(
		"http_apdex_frustrated_total",
		metric.WithDescription("Apdex: frustrated requests"),
		metric.WithUnit("{request}"),
	)
	if fruErr != nil {
		return nil, fruErr
	}

	return &HTTPMetrics{
		RequestCount:    requestCount,
		RequestDuration: requestDuration,
		SlowRequests:    slowRequests,
		ApdexSatisfied:  apdexSatisfied,
		ApdexTolerating: apdexTolerating,
		ApdexFrustrated: apdexFrustrated,
		Thresholds:      DefaultApdexThresholds(),
	}, nil
}

// RecordRequest records an HTTP request's metrics including duration and Apdex.
func (m *HTTPMetrics) RecordRequest(ctx context.Context, method, route string, statusCode int, duration time.Duration) {
	if m == nil {
		return
	}

	attrs := metric.WithAttributes(
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", statusCode),
	)

	m.RequestCount.Add(ctx, 1, attrs)
	m.RequestDuration.Record(ctx, duration.Seconds(), attrs)

	// Apdex classification
	routeAttrs := metric.WithAttributes(
		attribute.String("http.route", route),
	)

	switch {
	case duration <= m.Thresholds.Satisfied:
		m.ApdexSatisfied.Add(ctx, 1, routeAttrs)
	case duration <= m.Thresholds.Tolerating:
		m.ApdexTolerating.Add(ctx, 1, routeAttrs)
	default:
		m.ApdexFrustrated.Add(ctx, 1, routeAttrs)
		m.SlowRequests.Add(ctx, 1, routeAttrs)
	}
}
