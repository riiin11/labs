package trip

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	sq "github.com/Masterminds/squirrel"

	"github.com/you/trip-service/internal/domain/trip"
)

func (r *Repo) Save(ctx context.Context, t trip.Trip) error {
	sql, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_lat", "start_lon",
			"end_lat", "end_lon", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartPoint.Latitude, t.StartPoint.Longitude,
			t.EndPoint.Latitude, t.EndPoint.Longitude, t.Price, string(t.Status), t.StartedAt).
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
	sql, args, err := sq.Select("id", "user_id", "driver_id", "start_lat", "start_lon",
		"end_lat", "end_lon", "price", "status", "started_at", "finished_at", "last_position_at").
		From("trips").Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return trip.Trip{}, err
	}
	row := r.executor(ctx).QueryRow(ctx, sql, args...)
	return scanTrip(row)
}

func (r *Repo) Finish(ctx context.Context, id string, at time.Time) (trip.Trip, error) {
	sql, args, err := sq.Update("trips").
		Set("status", string(trip.StatusCompleted)).
		Set("finished_at", at).
		Where(sq.Eq{"id": id, "status": string(trip.StatusActive)}).
		Suffix("RETURNING id, user_id, driver_id, start_lat, start_lon, end_lat, end_lon, price, status, started_at, finished_at, last_position_at").
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return trip.Trip{}, err
	}
	row := r.executor(ctx).QueryRow(ctx, sql, args...)
	done, err := scanTrip(row)
	if err == nil {
		return done, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, err
	}
	// ноль строк: либо нет поездки, либо уже завершена — различаем
	check, args2, _ := sq.Select("status").From("trips").Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).ToSql()
	var status string
	if err2 := r.executor(ctx).QueryRow(ctx, check, args2...).Scan(&status); err2 != nil {
		if errors.Is(err2, pgx.ErrNoRows) {
			return trip.Trip{}, trip.ErrTripNotFound
		}
		return trip.Trip{}, err2
	}
	return trip.Trip{}, trip.ErrTripCompleted
}

func scanTrip(row pgx.Row) (trip.Trip, error) {
	var t trip.Trip
	var status string
	err := row.Scan(&t.ID, &t.UserID, &t.DriverID, &t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude, &t.Price, &status, &t.StartedAt,
		&t.FinishedAt, &t.LastPositionAt)
	if err != nil {
		return trip.Trip{}, err
	}
	t.Status = trip.Status(status)
	return t, nil
}