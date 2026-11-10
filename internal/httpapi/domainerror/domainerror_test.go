package domainerror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/you/trip-service/internal/domain/trip"
	"github.com/you/trip-service/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		status   int
		code     string
		noDetail string
	}{
		{"validation", trip.ValidationError{Field: "price", Message: "must be >= 0"},
			http.StatusBadRequest, "invalid_request", ""},
		{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found", ""},
		{"completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed", ""},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy", ""},
		{"internal", errors.New("secret db error"),
			http.StatusInternalServerError, "internal_error", "secret db error"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, p := domainerror.Problem(c.err, "/api/v1/trips")
			require.Equal(t, c.status, status)
			require.Equal(t, c.code, p.Code)
			require.Equal(t, c.status, p.Status)
			require.Equal(t, "/api/v1/trips", p.Instance)
			if c.noDetail != "" {
				require.NotContains(t, p.Detail, c.noDetail)
			}
		})
	}
}