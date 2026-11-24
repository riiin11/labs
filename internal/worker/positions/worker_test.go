package positions_test

import (
 "context"
 "errors"
 "log/slog"
 "strconv"
 "sync"
 "sync/atomic"
 "testing"
 "time"

 "github.com/stretchr/testify/require"
 "go.uber.org/goleak"

 "github.com/riiin11/labs/internal/domain/trip"
 "github.com/riiin11/labs/internal/worker/positions"
)

func TestMain(m *testing.M) {
 goleak.VerifyTestMain(m)
}

// --- Моки ---

type mockRepo struct {
 trips []trip.Trip
}

func (m *mockRepo) GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error) {
 return m.trips, nil
}

type infiniteRepo struct{}

func (m *infiniteRepo) GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error) {
 trips := make([]trip.Trip, 100)
 for i := 0; i < 100; i++ {
  trips[i] = trip.Trip{ID: "trip-" + strconv.Itoa(i)}
 }
 return trips, nil
}

type mockMetrics struct {
 staleCheckDropped int64
 staleCheckQueued  int64
 staleCheckDup     int64
}

func (m *mockMetrics) IncStaleCheck(ctx context.Context, result string) {
 switch result {
 case "dropped":
  atomic.AddInt64(&m.staleCheckDropped, 1)
 case "queued":
  atomic.AddInt64(&m.staleCheckQueued, 1)
 case "duplicate":
  atomic.AddInt64(&m.staleCheckDup, 1)
 }
}
func (m *mockMetrics) IncQueueSize(ctx context.Context, delta int64)           {}
func (m *mockMetrics) IncWorkersActive(ctx context.Context, delta int64)      {}
func (m *mockMetrics) ObservePositionRequest(ctx context.Context, r string, d time.Duration) {}

type mockGateway struct {
 mu    sync.Mutex
 calls []string
 delay time.Duration
 err   error
}

func (m *mockGateway) SendRequestPosition(ctx context.Context, driverID, tripID string) error {
 if m.delay > 0 {
  select {
  case <-time.After(m.delay):
  case <-ctx.Done():
   return ctx.Err()
  }
 }
 m.mu.Lock()
 m.calls = append(m.calls, tripID)
 m.mu.Unlock()
 return m.err
}

func (m *mockGateway) CallCount() int {
 m.mu.Lock()
 defer m.mu.Unlock()
 return len(m.calls)
}

type blockingGateway struct {
 block chan struct{}
}

func (m *blockingGateway) SendRequestPosition(ctx context.Context, driverID, tripID string) error {
 select {
 case <-m.block:
  return nil
 case <-ctx.Done():
  return ctx.Err()
 }
}

// --- Тесты ---

func TestWorkerPool_QueueOverflow(t *testing.T) {
 repo := &infiniteRepo{}
 
 blockChan := make(chan struct{})
 gw := &blockingGateway{block: blockChan}
 m := &mockMetrics{}
 
 cfg := positions.Config{
  CheckInterval: 10 * time.Millisecond,
  StaleAfter:    time.Minute,
  Workers:       1, 
  QueueSize:     1, 
 }
 
 pool := positions.NewWorkerPool(repo, gw, m, cfg, slog.Default())
 ctx, cancel := context.WithCancel(context.Background())
 defer cancel()
 
 pool.Start(ctx)
 
 // Даем сканеру время сделать много тиков
 time.Sleep(200 * time.Millisecond) 
 
 require.Greater(t, atomic.LoadInt64(&m.staleCheckDropped), int64(0), "expected dropped metric to be > 0")
 require.Greater(t, atomic.LoadInt64(&m.staleCheckQueued), int64(0), "expected queued metric to be > 0")
 
 close(blockChan) 
 cancel()
 pool.Stop(context.Background())
}

func TestWorkerPool_Deduplication(t *testing.T) {
 repo := &mockRepo{
  trips: []trip.Trip{{ID: "1"}}, 
 }
 gw := &mockGateway{delay: 50 * time.Millisecond}
 m := &mockMetrics{}
 
 cfg := positions.Config{
  CheckInterval: 10 * time.Millisecond,
  StaleAfter:    time.Minute,
  Workers:       1,
  QueueSize:     10,
 }
 
 pool := positions.NewWorkerPool(repo, gw, m, cfg, slog.Default())
 ctx, cancel := context.WithCancel(context.Background())
 defer cancel()
 
 pool.Start(ctx)
 time.Sleep(50 * time.Millisecond)
 pool.Stop(context.Background())
 
 require.LessOrEqual(t, gw.CallCount(), 2, "trip should not be duplicated in queue")
}

func TestWorkerPool_Cancel(t *testing.T) {
 repo := &mockRepo{
  trips: []trip.Trip{{ID: "1"}},
 }
 gw := &mockGateway{delay: 50 * time.Millisecond}
 m := &mockMetrics{}
 
 cfg := positions.Config{
  CheckInterval: 10 * time.Millisecond,
  StaleAfter:    time.Minute,
  Workers:       1,
  QueueSize:     10,
 }
 
 pool := positions.NewWorkerPool(repo, gw, m, cfg, slog.Default())
 ctx, cancel :=context.WithCancel(context.Background())
 defer cancel()
 
 pool.Start(ctx)
 time.Sleep(20 * time.Millisecond)
 
 pool.Cancel("1")
 
 time.Sleep(30 * time.Millisecond)
 pool.Stop(context.Background())
 
 require.Equal(t, 0, gw.CallCount(), "push service should not be called for cancelled trip")
}

func TestWorkerPool_PushErrorDoesNotCrash(t *testing.T) {
 repo := &mockRepo{
  trips: []trip.Trip{{ID: "1"}},
 }
 gw := &mockGateway{
  delay: 10 * time.Millisecond,
  err:   errors.New("push failed"),
 }
 m := &mockMetrics{}
 
 cfg := positions.Config{
  CheckInterval: 10 * time.Millisecond,
  StaleAfter:    time.Minute,
  Workers:       1,
  QueueSize:     10,
 }
 
 pool := positions.NewWorkerPool(repo, gw, m, cfg, slog.Default())
 ctx, cancel := context.WithCancel(context.Background())
 defer cancel()
 
 pool.Start(ctx)
 time.Sleep(50 * time.Millisecond)
 pool.Stop(context.Background())
}