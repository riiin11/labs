package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/riiin11/labs/internal/domain/trip"
)

type Repo interface {
	GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error)
}

type PushGateway interface {
	SendRequestPosition(ctx context.Context, driverID, tripID string) error
}

type Metrics interface {
	IncStaleCheck(ctx context.Context, result string)
	IncQueueSize(ctx context.Context, delta int64)
	IncWorkersActive(ctx context.Context, delta int64)
	ObservePositionRequest(ctx context.Context, result string, dur time.Duration)
}

type WorkerPool struct {
	repo    Repo
	gateway PushGateway
	metrics Metrics
	cfg     Config
	log     *slog.Logger

	queue    chan trip.Trip
	inFlight map[string]bool
	mu       sync.Mutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

type Config struct {
	CheckInterval time.Duration
	StaleAfter    time.Duration
	Workers       int
	QueueSize     int
}

func NewWorkerPool(repo Repo, gw PushGateway, m Metrics, cfg Config, log *slog.Logger) *WorkerPool {
	return &WorkerPool{
		repo:     repo,
		gateway:  gw,
		metrics:  m,
		cfg:      cfg,
		log:      log,
		queue:    make(chan trip.Trip, cfg.QueueSize),
		inFlight: make(map[string]bool),
	}
}

func (w *WorkerPool) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	// 1. Запускаем воркеров
	for i := 0; i < w.cfg.Workers; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}

	w.wg.Add(1)
	go w.scanner(ctx)
}

func (w *WorkerPool) Stop(ctx context.Context) {
	w.cancel()
	close(w.queue)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.log.Info("worker pool stopped gracefully")
	case <-ctx.Done():
		w.log.Warn("worker pool stop timeout")
	}
}

func (w *WorkerPool) Cancel(tripID string) {
	w.mu.Lock()
	delete(w.inFlight, tripID)
	w.mu.Unlock()
}

func (w *WorkerPool) scanner(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.scan(ctx)
		}
	}
}

func (w *WorkerPool) scan(ctx context.Context) {
	trips, err := w.repo.GetStaleTrips(ctx, w.cfg.StaleAfter)
	if err != nil {
		w.log.ErrorContext(ctx, "failed to fetch stale trips", "err", err)
		return
	}

	for _, t := range trips {
		w.mu.Lock()
		if w.inFlight[t.ID] {
			w.mu.Unlock()
			w.metrics.IncStaleCheck(ctx, "duplicate")
			continue
		}
		w.inFlight[t.ID] = true
		w.mu.Unlock()

		select {
		case w.queue <- t:
			w.metrics.IncStaleCheck(ctx, "queued")
			w.metrics.IncQueueSize(ctx, 1)
		default:
			w.mu.Lock()
			delete(w.inFlight, t.ID)
			w.mu.Unlock()

			w.metrics.IncStaleCheck(ctx, "dropped")
			w.log.WarnContext(ctx, "queue full, dropping stale check", "trip_id", t.ID)
		}
	}
}

func (w *WorkerPool) worker(ctx context.Context, id int) {
	defer w.wg.Done()
	w.log.Info("worker started", "id", id)

	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-w.queue:
			if !ok {
				return
			}
			w.metrics.IncQueueSize(ctx, -1)
			w.process(ctx, t)
		}
	}
}

func (w *WorkerPool) process(ctx context.Context, t trip.Trip) {
	w.metrics.IncWorkersActive(ctx, 1)
	defer w.metrics.IncWorkersActive(ctx, -1)

	w.mu.Lock()
	active := w.inFlight[t.ID]
	w.mu.Unlock()

	// Если хендлер уже отменил задачу (пришла координата)
	if !active {
		return
	}

	tracer := otel.Tracer("position-worker")
	ctx, span := tracer.Start(ctx, "push.request_position", trace.WithAttributes())
	defer span.End()

	start := time.Now()
	err := w.gateway.SendRequestPosition(ctx, t.DriverID, t.ID)
	dur := time.Since(start)

	if err != nil {
		w.metrics.ObservePositionRequest(ctx, "error", dur)
		w.log.ErrorContext(ctx, "push request failed", "err", err, "trip_id", t.ID)
	} else {
		w.metrics.ObservePositionRequest(ctx, "success", dur)
	}

	// Завершили обработку — убираем из очереди
	w.mu.Lock()
	delete(w.inFlight, t.ID)
	w.mu.Unlock()
}
