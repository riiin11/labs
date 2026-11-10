package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/riiin11/labs/internal/generated/api"
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

					detail := "Something went wrong"
					instance := r.URL.Path

					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{
						Type:     "https://tripgo.example/problems/internal-error",
						Title:    "Internal error",
						Status:   http.StatusInternalServerError,
						Detail:   &detail,
						Instance: &instance,
						Code:     "internal_error",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}