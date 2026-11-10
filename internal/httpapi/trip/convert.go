package trip

import (
	"github.com/you/trip-service/internal/domain/trip"
	"github.com/you/trip-service/internal/generated/api"
)

func toInput(r api.CreateTripRequest) trip.CreateInput {
	return trip.CreateInput{
		UserID:     r.UserId.String(),
		DriverID:   r.DriverId.String(),
		StartPoint: trip.Point{Latitude: r.StartPoint.Latitude, Longitude: r.StartPoint.Longitude},
		EndPoint:   trip.Point{Latitude: r.EndPoint.Latitude, Longitude: r.EndPoint.Longitude},
		Price:      r.Price,
	}
}

func toResponse(t trip.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Point{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:       api.Point{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}