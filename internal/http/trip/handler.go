package trip

import (
	"encoding/json"
	"net/http"
	"time"
	"context"

	domaintrip "github.com/riiin11/labs/internal/domain/trip"
	"github.com/riiin11/labs/internal/generated/api"
	"github.com/riiin11/labs/internal/http/domainerror"
)

type Handler struct {
	create       createUsecase
	get          getUsecase
	finish       finishUsecase
	savePosition savePositionUsecase
	getPositions getPositionsUsecase
	pool         pinger
}

func New(
	create createUsecase,
	get getUsecase,
	finish finishUsecase,
	savePos savePositionUsecase,
	getPos getPositionsUsecase,
	pool pinger,
) *Handler {
	return &Handler{
		create:       create,
		get:          get,
		finish:       finish,
		savePosition: savePos,
		getPositions: getPos,
		pool:         pool,
	}
}

var _ api.ServerInterface = (*Handler)(nil)

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var req api.TripData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		status, body := domainerror.Problem(
			domainerror.Validation("body", "invalid JSON"), r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	t, err := h.create.Create(r.Context(), toInput(req))
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+t.ID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toResponse(t))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	t, err := h.get.Get(r.Context(), tripId.String())
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	t, err := h.finish.Finish(r.Context(), tripId.String())
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(t))
}

func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	var req api.PositionData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		status, body := domainerror.Problem(
			domainerror.Validation("body", "invalid JSON"), r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	p := domaintrip.Position{
		TripID:     tripId.String(),
		Latitude:   req.Coordinates.Latitude,
		Longitude:  req.Coordinates.Longitude,
		RecordedAt: req.RecordedAt,
	}

	saved, err := h.savePosition.Save(r.Context(), p)
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	resp := api.TripPosition{
		Id:     saved.ID,
		TripId: tripId, 
		Coordinates: api.Coordinates{
			Latitude:  saved.Latitude,
			Longitude: saved.Longitude,
		},
		RecordedAt: saved.RecordedAt,
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	positions, err := h.getPositions.Get(r.Context(), tripId.String())
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	resp := api.TripPositionsResponse{
		Positions: make([]api.TripPosition, 0, len(positions)),
	}
	
	for _, p := range positions {
		resp.Positions = append(resp.Positions, api.TripPosition{
			Id:     p.ID,
			TripId: tripId,
			Coordinates: api.Coordinates{
				Latitude:  p.Latitude,
				Longitude: p.Longitude,
			},
			RecordedAt: p.RecordedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if h.pool == nil || h.pool.Ping(ctx) != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Unavailable})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, p api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}