package trip

import (
	"context"
	"errors"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riiin11/labs/internal/domain/trip"
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

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
}

func statusPtr(s trip.Status) any {
	if s == "" {
		return nil
	}
	return string(s)
}

func (r *Repo) Save(ctx context.Context, t trip.Trip) error {
	q, args, err := sq.Update("trips").
		Set("status", t.Status).
		Set("finished_at", t.FinishedAt).
		Where(sq.Eq{"id": t.ID, "status": trip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	tag, err := r.executor(ctx).Exec(ctx, q, args...)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return trip.ErrTripCompleted
	}
	return nil
}

func (r *Repo) SaveStatusHistory(ctx context.Context, tripID string, from, to trip.Status) error {
	sql, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "changed_at").
		Values(uuid.MustParse(tripID), statusPtr(from), string(to), time.Now().UTC()).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	return err
}

func (r *Repo) GetByID(ctx context.Context, id string) (trip.Trip, error) {
	sql, args, err := sq.Select(tripColumns...).
		From("trips").
		Where(sq.Eq{"id": uuid.MustParse(id)}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Trip{}, err
	}

	row := r.executor(ctx).QueryRow(ctx, sql, args...)
	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, trip.ErrTripNotFound
	}
	return t, err
}