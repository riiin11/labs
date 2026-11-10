package middleware

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func Metrics() func(http.Handler) http.Handler {
	mp := otel.GetMeterProvider()
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http.server",
			otelhttp.WithMeterProvider(mp),
		)
	}
}