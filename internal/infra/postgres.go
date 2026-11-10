package infra

import (
	"context"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/you/trip-service/internal/config"
)

func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	pc, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	pc.MaxConns = cfg.DBMaxConns
	pc.MinConns = cfg.DBMinConns
	pc.MaxConnLifetime = cfg.DBMaxConnLifetime
	pc.ConnConfig.Tracer = otelpgx.NewTracer()

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, pc)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}