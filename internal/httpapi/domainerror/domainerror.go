package domainerror

import (
	"errors"
	"net/http"

	"github.com/you/trip-service/internal/domain/trip"
	"github.com/you/trip-service/internal/generated/api"
)

func Validation(field, msg string) error {
	return trip.ValidationError{Field: field, Message: msg}
}

func Problem(err error, instance string) (int, api.Problem) {
	var ve trip.ValidationError
	switch {
	case errors.As(err, &ve):
		return http.StatusBadRequest, mk(http.StatusBadRequest, "invalid_request",
			"Invalid request", ve.Error(), instance)
	case errors.Is(err, trip.ErrTripNotFound):
		return http.StatusNotFound, mk(http.StatusNotFound, "trip_not_found",
			"Trip not found", "Trip with this id does not exist", instance)
	case errors.Is(err, trip.ErrTripCompleted):
		return http.StatusConflict, mk(http.StatusConflict, "trip_completed",
			"Trip completed", "Trip is already completed", instance)
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, mk(http.StatusConflict, "driver_busy",
			"Driver busy", "Driver already has an active trip", instance)
	default:
		return http.StatusInternalServerError, mk(http.StatusInternalServerError,
			"internal_error", "Internal error", "Something went wrong", instance)
	}
}

func mk(status int, code, title, detail, instance string) api.Problem {
	return api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Code:     code,
		Instance: instance,
	}
}