package trip

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/you/trip-service/internal/generated/api"
	"github.com/you/trip-service/internal/http/domainerror"
)

type CreateHandler struct{ uc createUsecase }

func NewCreate(uc createUsecase) *CreateHandler { return &CreateHandler{uc: uc} }

func (h *CreateHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		status, body := domainerror.Problem(
			domainerror.Validation("body", "invalid JSON"), r.URL.Path)
		writeProblem(w, status, body)
		return
	}

	t, err := h.uc.Create(r.Context(), toInput(req))
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

type GetHandler struct{ uc getUsecase }

func NewGet(uc getUsecase) *GetHandler { return &GetHandler{uc: uc} }

func (h *GetHandler) Handle(w http.ResponseWriter, r *http.Request) {
	t, err := h.uc.Get(r.Context(), chi.URLParam(r, "tripId"))
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(t))
}

type FinishHandler struct{ uc finishUsecase }

func NewFinish(uc finishUsecase) *FinishHandler { return &FinishHandler{uc: uc} }

func (h *FinishHandler) Handle(w http.ResponseWriter, r *http.Request) {
	t, err := h.uc.Finish(r.Context(), chi.URLParam(r, "tripId"))
	if err != nil {
		status, body := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, body)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(t))
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