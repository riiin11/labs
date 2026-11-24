package trip_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/riiin11/labs/internal/domain/trip"
	"github.com/riiin11/labs/internal/generated/api"
	triphttp "github.com/riiin11/labs/internal/http/trip"
)

var errBoom = errors.New("secret db error")

func TestCreateHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockcreateUsecase(ctrl)

	done := trip.Trip{
		ID:         uuid.NewString(),
		UserID:     uuid.NewString(),
		DriverID:   uuid.NewString(),
		Status:     trip.StatusActive,
		Price:      1450,
	}
	uc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(done, nil)

	h := triphttp.New(uc, nil, nil, nil)
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9,"longitude":30.3},"end_point":{"latitude":59.8,"longitude":30.4},"price":1450}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req, api.CreateTripParams{})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "/api/v1/trips/"+done.ID, rec.Header().Get("Location"))
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestCreateHandler_BadJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockcreateUsecase(ctrl)

	h := triphttp.New(uc, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	h.CreateTrip(rec, req, api.CreateTripParams{})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "invalid_request", p.Code)
}

func TestGetHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockgetUsecase(ctrl)
	id := uuid.NewString()
	uc.EXPECT().Get(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripNotFound)

	h := triphttp.New(nil, uc, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id, nil)
	rec := httptest.NewRecorder()

	h.GetTrip(rec, req, api.TripId(uuid.MustParse(id)))

	require.Equal(t, http.StatusNotFound, rec.Code)
	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "trip_not_found", p.Code)
}

func TestFinishHandler_Completed(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockfinishUsecase(ctrl)
	id := uuid.NewString()
	uc.EXPECT().Finish(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripCompleted)

	h := triphttp.New(nil, nil, uc, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id+"/finish", nil)
	rec := httptest.NewRecorder()

	h.FinishTrip(rec, req, api.TripId(uuid.MustParse(id)))

	require.Equal(t, http.StatusConflict, rec.Code)
	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "trip_completed", p.Code)
}

func TestFinishHandler_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockfinishUsecase(ctrl)
	id := uuid.NewString()
	uc.EXPECT().Finish(gomock.Any(), id).Return(trip.Trip{}, errBoom)

	h := triphttp.New(nil, nil, uc, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id+"/finish", nil)
	rec := httptest.NewRecorder()

	h.FinishTrip(rec, req, api.TripId(uuid.MustParse(id)))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "secret db error")
}