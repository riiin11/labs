package trip

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/riiin11/labs/internal/generated/api"
	"github.com/riiin11/labs/internal/http/domainerror"
)

type pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	create createUsecase
	get    getUsecase
	finish finishUsecase
	pool   pinger
}

func New(create createUsecase, get getUsecase, finish finishUsecase, pool pinger) *Handler {
	return &Handler{create: create, get: get, finish: finish, pool: pool}
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

func (h *Handler) ListTripPositions(w http.ResponseWriter, r *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handler) CreateTripPosition(w http.ResponseWriter, r *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
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