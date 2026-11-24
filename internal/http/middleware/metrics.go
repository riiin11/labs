package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var activeRequests, _ = otel.Meter("http.server").Int64UpDownCounter(
	"http.server.active_requests",
	metric.WithDescription("Number of requests currently in flight"),
)

func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		withActive := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			activeRequests.Add(r.Context(), 1)
			defer activeRequests.Add(r.Context(), -1)

			next.ServeHTTP(w, r)
		})

		return otelhttp.NewHandler(withActive, "http.server",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = r.URL.Path
				}

				return r.Method + " " + route
			}),
		)
	}
}