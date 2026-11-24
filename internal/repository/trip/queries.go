package trip

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/riiin11/labs/internal/domain/trip"
)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
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

func statusPtr(s trip.Status) any {
	if s == "" {
		return nil
	}

	return string(s)
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

func (r *Repo) Finish(ctx context.Context, id string, at time.Time) (trip.Trip, error) {
	sql, args, err := sq.Update("trips").
		Set("status", string(trip.StatusCompleted)).
		Set("finished_at", at).
		Set("updated_at", at).
		Where(sq.Eq{"id": uuid.MustParse(id), "status": string(trip.StatusActive)}).
		Suffix("RETURNING " + returningColumns()).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Trip{}, err
	}

	done, err := scanTrip(r.executor(ctx).QueryRow(ctx, sql, args...))
	if err == nil {
		return done, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, err
	}

	checkSQL, checkArgs, _ := sq.Select("status").
		From("trips").
		Where(sq.Eq{"id": uuid.MustParse(id)}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	var status string

	if err := r.executor(ctx).QueryRow(ctx, checkSQL, checkArgs...).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return trip.Trip{}, trip.ErrTripNotFound
		}

		return trip.Trip{}, err
	}

	return trip.Trip{}, trip.ErrTripCompleted
}

func returningColumns() string {
	out := ""

	for i, c := range tripColumns {
		if i > 0 {
			out += ", "
		}

		out += c
	}

	return out
}