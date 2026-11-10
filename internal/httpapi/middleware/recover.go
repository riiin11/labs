package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/you/trip-service/internal/generated/api"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{
						Type:     "https://tripgo.example/problems/internal-error",
						Title:    "Internal error",
						Status:   http.StatusInternalServerError,
						Detail:   "Something went wrong",
						Code:     "internal_error",
						Instance: r.URL.Path,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}