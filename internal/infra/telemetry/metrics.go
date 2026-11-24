package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/attribute"
)

type Metrics struct {
	TripsCreated   metric.Int64Counter
	TripsCompleted metric.Int64Counter

	// Новые для ЛР3
	PositionsSaved          metric.Int64Counter
	StaleChecks             metric.Int64Counter
	QueueSize               metric.Int64UpDownCounter
	WorkersActive           metric.Int64UpDownCounter
	PositionRequests        metric.Int64Counter
	PositionRequestDuration metric.Float64Histogram
}

func NewMetrics() *Metrics {
	m := otel.Meter("trip-service")

	created, _ := m.Int64Counter("tripgo.trips.created")
	completed, _ := m.Int64Counter("tripgo.trips.completed")

	posSaved, _ := m.Int64Counter("tripgo.positions.saved")
	staleChecks, _ := m.Int64Counter("tripgo.stale_checks")
	queueSize, _ := m.Int64UpDownCounter("tripgo.position_requests.queue.size")
	workersActive, _ := m.Int64UpDownCounter("tripgo.position_requests.workers.active")
	posRequests, _ := m.Int64Counter("tripgo.position_requests")
	posDuration, _ := m.Float64Histogram("tripgo.position_requests.duration")

	return &Metrics{
		TripsCreated:   created,
		TripsCompleted: completed,

		PositionsSaved:          posSaved,
		StaleChecks:             staleChecks,
		QueueSize:               queueSize,
		WorkersActive:           workersActive,
		PositionRequests:        posRequests,
		PositionRequestDuration: posDuration,
	}
}

func (m *Metrics) IncCreated(ctx context.Context) {
	if m.TripsCreated != nil {
		m.TripsCreated.Add(ctx, 1)
	}
}

func (m *Metrics) IncCompleted(ctx context.Context) {
	if m.TripsCompleted != nil {
		m.TripsCompleted.Add(ctx, 1)
	}
}

func (m *Metrics) IncPositionsSaved(ctx context.Context) {
	if m.PositionsSaved != nil {
		m.PositionsSaved.Add(ctx, 1)
	}
}

func (m *Metrics) IncStaleCheck(ctx context.Context, result string) {
	if m.StaleChecks != nil {
		m.StaleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	}
}

func (m *Metrics) IncQueueSize(ctx context.Context, delta int64) {
	if m.QueueSize != nil {
		m.QueueSize.Add(ctx, delta)
	}
}

func (m *Metrics) IncWorkersActive(ctx context.Context, delta int64) {
	if m.WorkersActive != nil {
		m.WorkersActive.Add(ctx, delta)
	}
}

func (m *Metrics) ObservePositionRequest(ctx context.Context, result string, dur time.Duration) {
	if m.PositionRequestDuration != nil {
		m.PositionRequestDuration.Record(ctx, dur.Seconds(), metric.WithAttributes(attribute.String("result", result)))
	}
	if m.PositionRequests != nil {
		m.PositionRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	}
}