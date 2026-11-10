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

func (r *Repo) Save(ctx context.Context, t trip.Trip) error {
	sql, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_lat", "start_lon",
			"end_lat", "end_lon", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID,
			t.StartPoint.Latitude, t.StartPoint.Longitude,
			t.EndPoint.Latitude, t.EndPoint.Longitude,
			t.Price, string(t.Status), t.StartedAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	return mapInsertError(err)
}

func (r *Repo) SaveStatusHistory(ctx context.Context, tripID string, status trip.Status) error {
	sql, args, err := sq.Insert("trip_status_history").
		Columns("id", "trip_id", "status", "changed_at").
		Values(uuid.NewString(), tripID, string(status), time.Now().UTC()).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}
	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	return err
}

func (r *Repo) GetByID(ctx context.Context, id string) (trip.Trip, error) {
	sql, args, err := sq.Select("id", "user_id", "driver_id",
		"start_lat", "start_lon", "end_lat", "end_lon",
		"price", "status", "started_at", "finished_at", "last_position_at").
		From("trips").
		Where(sq.Eq{"id": id}).
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
		Where(sq.Eq{"id": id, "status": string(trip.StatusActive)}).
		Suffix("RETURNING id, user_id, driver_id, start_lat, start_lon, end_lat, end_lon, price, status, started_at, finished_at, last_position_at").
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
		Where(sq.Eq{"id": id}).
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