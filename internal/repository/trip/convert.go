package trip

import (
	"github.com/jackc/pgx/v5"

	"github.com/riiin11/labs/internal/domain/trip"
)

func scanTrip(row pgx.Row) (trip.Trip, error) {
	var t trip.Trip
	var status string
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &status, &t.StartedAt,
		&t.FinishedAt, &t.LastPositionAt,
	)
	if err != nil {
		return trip.Trip{}, err
	}
	t.Status = trip.Status(status)
	return t, nil
}