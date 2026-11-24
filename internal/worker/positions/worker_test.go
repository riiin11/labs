package positions_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/riiin11/labs/internal/worker/positions"
	"github.com/riiin11/labs/internal/domain/trip"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// Заглушки для интерфейсов
type mockRepo struct{}
func (m *mockRepo) GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error) { return nil, nil }

type mockGW struct{}
func (m *mockGW) SendRequestPosition(ctx context.Context, driverID, tripID string) error { return nil }

type mockMetrics struct{}
func (m *mockMetrics) IncStaleCheck(ctx context.Context, result string) {}
func (m *mockMetrics) IncQueueSize(ctx context.Context, delta int64) {}
func (m *mockMetrics) IncWorkersActive(ctx context.Context, delta int64) {}
func (m *mockMetrics) ObservePositionRequest(ctx context.Context, result string, dur time.Duration) {}

func TestWorkerPool_Stop(t *testing.T) {
	cfg := positions.Config{
		CheckInterval: 10 * time.Millisecond,
		StaleAfter:    time.Minute,
		Workers:       2,
		QueueSize:     10,
	}
	
	pool := positions.NewWorkerPool(&mockRepo{}, &mockGW{}, &mockMetrics{}, cfg, slog.Default())
	
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)
	
	// Даем воркерам поработать
	time.Sleep(50 * time.Millisecond)
	
	// Останавливаем
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	
	cancel() // отменяем контекст, чтобы сканер встал
	pool.Stop(shutdownCtx)
	
	// Если тест дошел сюда без ошибок и goleak молчит — все отлично!
}