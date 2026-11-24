package container

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riiin11/labs/internal/config"
	"github.com/riiin11/labs/internal/generated/api"
	"github.com/riiin11/labs/internal/http/domainerror"
	"github.com/riiin11/labs/internal/http/middleware"
	triphttp "github.com/riiin11/labs/internal/http/trip"
	"github.com/riiin11/labs/internal/infra"
	"github.com/riiin11/labs/internal/infra/telemetry"
	triprepo "github.com/riiin11/labs/internal/repository/trip"
	tripuc "github.com/riiin11/labs/internal/usecase/trip"
)

type App struct {
	cfg    config.Config
	log    *slog.Logger
	server *http.Server
	pool   *pgxpool.Pool
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
		_ = otelShutdown(ctx)
		return nil, err
	}

	tx := infra.NewTxManager(pool, cfg.DBQueryTimeout)
	repo := triprepo.New(pool, log)
	metrics := telemetry.NewMetrics()

	createUC := tripuc.NewCreate(repo, tx, metrics, log)
	getUC := tripuc.NewGet(repo)
	finishUC := tripuc.NewFinish(repo, tx, metrics, log)

	h := triphttp.New(createUC, getUC, finishUC, pool)

	r := chi.NewRouter()
	r.Use(middleware.Recover(log))
	r.Use(middleware.Logging(log))
	r.Use(middleware.Metrics())

	api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			status, body := domainerror.Problem(
				domainerror.Validation("trip_id", "must be a valid UUID"), r.URL.Path)
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		},
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