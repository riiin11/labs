package trip_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/you/trip-service/internal/domain/trip"
	triphttp "github.com/you/trip-service/internal/http/trip"
	"github.com/you/trip-service/internal/generated/api"
)

func TestCreateHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockcreateUsecase(ctrl)

	done := trip.Trip{ID: "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111", Status: trip.StatusActive}
	uc.EXPECT().Create(gomock.Any(), gomock.Any()).Return(done, nil)

	h := triphttp.NewCreate(uc)
	body := `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9,"longitude":30.3},"end_point":{"latitude":59.8,"longitude":30.4},"price":1450}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Handle(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "/api/v1/trips/"+done.ID, rec.Header().Get("Location"))
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestCreateHandler_BadJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockcreateUsecase(ctrl)

	h := triphttp.NewCreate(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	h.Handle(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "invalid_request", p.Code)
}

func TestGetHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockgetUsecase(ctrl)
	id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
	uc.EXPECT().Get(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripNotFound)

	h := triphttp.NewGet(uc)
	r := chi.NewRouter()
	r.Get("/api/v1/trips/{tripId}", h.Handle)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "trip_not_found", p.Code)
}

func TestFinishHandler_Completed(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockfinishUsecase(ctrl)
	id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
	uc.EXPECT().Finish(gomock.Any(), id).Return(trip.Trip{}, trip.ErrTripCompleted)

	h := triphttp.NewFinish(uc)
	r := chi.NewRouter()
	r.Post("/api/v1/trips/{tripId}/finish", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id+"/finish", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	var p api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p))
	require.Equal(t, "trip_completed", p.Code)
}

func TestFinishHandler_Internal(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockfinishUsecase(ctrl)
	id := "1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"
	uc.EXPECT().Finish(gomock.Any(), id).Return(trip.Trip{}, errBoom)

	h := triphttp.NewFinish(uc)
	r := chi.NewRouter()
	r.Post("/api/v1/trips/{tripId}/finish", h.Handle)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id+"/finish", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "secret")
}