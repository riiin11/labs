package container

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/you/trip-service/internal/config"
	"github.com/you/trip-service/internal/http"
	triphttp "github.com/you/trip-service/internal/http/trip"
	"github.com/you/trip-service/internal/http/middleware"
	"github.com/you/trip-service/internal/infra"
	"github.com/you/trip-service/internal/infra/telemetry"
	"github.com/you/trip-service/internal/repository/trip"
	"github.com/you/trip-service/internal/usecase/trip"
)

type App struct {
	cfg    config.Config
	log    *slog.Logger
	server *http.Server
	pool   interface{ Close() }
	otel   func(context.Context) error
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	log := telemetry.NewLogger(cfg.LogLevel)

	otelShutdown, err := telemetry.Init(ctx)
	if err != nil {
		log.Warn("otel init failed, continuing without telemetry", "err", err)
		otelShutdown = func(context.Context) error { return nil }
	}

	pool, err := infra.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tx := infra.NewTxManager(pool)
	repo := triprepo.New(pool)
	metrics := telemetry.NewMetrics()

	createUC := tripuc.NewCreate(repo, tx, metrics, log)
	getUC := tripuc.NewGet(repo)
	finishUC := tripuc.NewFinish(repo, tx, metrics, log)

	hCreate := triphttp.NewCreate(createUC)
	hGet := triphttp.NewGet(getUC)
	hFinish := triphttp.NewFinish(finishUC)

	r := chi.NewRouter()
	r.Use(middleware.Recover(log))
	r.Use(middleware.Tracing())
	r.Use(middleware.Logging(log))
	r.Use(middleware.Metrics())

	r.Get("/health", http.Health)
	r.Get("/ready", http.Ready(pool))

	r.Route("/api/v1/trips", func(r chi.Router) {
		r.Post("/", hCreate.Handle)
		r.Get("/{tripId}", hGet.Handle)
		r.Post("/{tripId}/finish", hFinish.Handle)
	})

	return &App{
		cfg:  cfg,
		log:  log,
		pool: pool,
		otel: otelShutdown,
		server: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           r,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 3 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- a.server.ListenAndServe() }()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()
	return a.server.Shutdown(shutdownCtx)
}

func (a *App) Close() {
	a.pool.Close()
	_ = a.otel(context.Background())
}