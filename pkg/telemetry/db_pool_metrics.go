package telemetry

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RegisterDBPoolMetrics registers observable gauges for SQL database pool stats.
// This provides visibility into connection pool health (open, in_use, idle, max_open, wait_count).
func RegisterDBPoolMetrics(_ context.Context, serviceName string, db *sql.DB, poolName string) error {
	if db == nil {
		return nil
	}

	meter := otel.Meter(serviceName)
	poolAttr := metric.WithAttributes(attribute.String("db.pool.name", poolName))

	_, openErr := meter.Int64ObservableGauge(
		"db_pool_open_connections",
		metric.WithDescription("Number of open connections in the pool"),
		metric.WithUnit("{connection}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			stats := db.Stats()
			o.Observe(int64(stats.OpenConnections), poolAttr)
			return nil
		}),
	)
	if openErr != nil {
		return openErr
	}

	_, inUseErr := meter.Int64ObservableGauge(
		"db_pool_in_use_connections",
		metric.WithDescription("Number of connections currently in use"),
		metric.WithUnit("{connection}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			stats := db.Stats()
			o.Observe(int64(stats.InUse), poolAttr)
			return nil
		}),
	)
	if inUseErr != nil {
		return inUseErr
	}

	_, idleErr := meter.Int64ObservableGauge(
		"db_pool_idle_connections",
		metric.WithDescription("Number of idle connections in the pool"),
		metric.WithUnit("{connection}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			stats := db.Stats()
			o.Observe(int64(stats.Idle), poolAttr)
			return nil
		}),
	)
	if idleErr != nil {
		return idleErr
	}

	_, maxErr := meter.Int64ObservableGauge(
		"db_pool_max_open_connections",
		metric.WithDescription("Maximum number of open connections configured"),
		metric.WithUnit("{connection}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			stats := db.Stats()
			o.Observe(int64(stats.MaxOpenConnections), poolAttr)
			return nil
		}),
	)
	if maxErr != nil {
		return maxErr
	}

	_, waitErr := meter.Int64ObservableGauge(
		"db_pool_wait_count",
		metric.WithDescription("Total number of connections waited for"),
		metric.WithUnit("{connection}"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			stats := db.Stats()
			o.Observe(stats.WaitCount, poolAttr)
			return nil
		}),
	)
	if waitErr != nil {
		return waitErr
	}

	return nil
}
