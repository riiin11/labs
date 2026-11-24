package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewMetrics() *Metrics {
	m := otel.Meter("trip-service")
	c, _ := m.Int64Counter("tripgo.trips.created")
	d, _ := m.Int64Counter("tripgo.trips.completed")
	return &Metrics{created: c, completed: d}
}

func (m *Metrics) IncCreated(ctx context.Context)   { m.created.Add(ctx, 1) }
func (m *Metrics) IncCompleted(ctx context.Context) { m.completed.Add(ctx, 1) }