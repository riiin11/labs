package trip

import (
 "context"
 "log/slog"

 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgconn"
 "github.com/jackc/pgx/v5/pgxpool"

 "github.com/riiin11/labs/internal/infra"
)

type Repo struct {
 pool *pgxpool.Pool
 log  *slog.Logger
}

func New(pool *pgxpool.Pool, log *slog.Logger) *Repo {
 return &Repo{pool: pool, log: log}
}

type dbExecutor interface {
 QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
 Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
 Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *Repo) executor(ctx context.Context) dbExecutor {
 if tx, ok := infra.TxFromContext(ctx); ok {
  return tx
 }
 return r.pool
}