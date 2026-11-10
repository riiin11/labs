package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/you/trip-service/internal/generated/api"
)

type pinger interface {
	Ping(ctx context.Context) error
}

func Health(w http.ResponseWriter, r *http.Request) {
	writeHealth(w, http.StatusOK, api.HealthResponse{Status: "ok"})
}

func Ready(p pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			writeHealth(w, http.StatusServiceUnavailable, api.HealthResponse{Status: "unavailable"})
			return
		}
		writeHealth(w, http.StatusOK, api.HealthResponse{Status: "ok"})
	}
}

func writeHealth(w http.ResponseWriter, status int, h api.HealthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(h)
}