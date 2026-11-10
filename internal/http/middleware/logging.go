package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}
			sc := trace.SpanFromContext(r.Context()).SpanContext()

			attrs := []any{
				"method", r.Method,
				"route", route,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"trace_id", sc.TraceID().String(),
				"span_id", sc.SpanID().String(),
			}
			if sw.status >= 500 {
				log.Error("request handled", attrs...)
			} else {
				log.Info("request handled", attrs...)
			}
		})
	}
}