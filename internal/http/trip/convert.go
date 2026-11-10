package trip

import (
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/riiin11/labs/internal/domain/trip"
	"github.com/riiin11/labs/internal/generated/api"
)

func toInput(r api.TripData) trip.CreateInput {
	return trip.CreateInput{
		UserID:     r.UserId.String(),
		DriverID:   r.DriverId.String(),
		StartPoint: trip.Point{Latitude: r.StartPoint.Latitude, Longitude: r.StartPoint.Longitude},
		EndPoint:   trip.Point{Latitude: r.EndPoint.Latitude, Longitude: r.EndPoint.Longitude},
		Price:      r.Price,
	}
}

func toResponse(t trip.Trip) api.Trip {
	id, _ := uuid.Parse(t.ID)
	userID, _ := uuid.Parse(t.UserID)
	driverID, _ := uuid.Parse(t.DriverID)
	return api.Trip{
		Id:             openapi_types.UUID(id),
		UserId:         openapi_types.UUID(userID),
		DriverId:       openapi_types.UUID(driverID),
		StartPoint:     api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:       api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}