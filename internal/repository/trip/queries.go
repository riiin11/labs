package trip

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/riiin11/labs/internal/domain/trip"
)

func (r *Repo) SavePosition(ctx context.Context, p trip.Position) (trip.Position, error) {
	// 1. Вставляем координату
	insertQ, insertArgs, err := sq.Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(p.TripID, p.Latitude, p.Longitude, p.RecordedAt).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Position{}, err
	}

	var id int64
	err = r.executor(ctx).QueryRow(ctx, insertQ, insertArgs...).Scan(&id)
	if err != nil {
		return trip.Position{}, err
	}
	p.ID = id

	// 2. Обновляем last_position_at с GREATEST
	updateQ, updateArgs, err := sq.Update("trips").
		Set("last_position_at", sq.Expr("GREATEST(last_position_at, ?)", p.RecordedAt)).
		Where(sq.Eq{"id": p.TripID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Position{}, err
	}

	_, err = r.executor(ctx).Exec(ctx, updateQ, updateArgs...)
	if err != nil {
		return trip.Position{}, err
	}

	return p, nil
}

func (r *Repo) GetPositions(ctx context.Context, tripID string) ([]trip.Position, error) {
	q, args, err := sq.Select("id", "trip_id", "latitude", "longitude", "recorded_at").
		From("trip_positions").
		Where(sq.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.executor(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []trip.Position
	for rows.Next() {
		var p trip.Position
		if err := rows.Scan(&p.ID, &p.TripID, &p.Latitude, &p.Longitude, &p.RecordedAt); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

func (r *Repo) GetStaleTrips(ctx context.Context, staleAfter time.Duration) ([]trip.Trip, error) {
	threshold := time.Now().UTC().Add(-staleAfter)

	q, args, err := sq.Select("id", "driver_id").
		From("trips").
		Where(sq.Eq{"status": trip.StatusActive}).
		Where("COALESCE(last_position_at, started_at) < ?", threshold).
		OrderBy("COALESCE(last_position_at, started_at) ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.executor(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []trip.Trip
	for rows.Next() {
		var t trip.Trip
		if err := rows.Scan(&t.ID, &t.DriverID); err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}