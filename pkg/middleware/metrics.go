package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/vsayfb/gig-platform-core-service/pkg/metrics"
)

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		
		if route == "" {
			route = "unknown"
		}

		status := strconv.Itoa(rw.status)
		duration := time.Since(start).Seconds()

		attrs := metric.WithAttributes(
			attribute.String("route", route),
			attribute.String("method", r.Method),
			attribute.String("status", status),
		)

		metrics.HttpRequestsTotal.Add(r.Context(), 1, attrs)
		metrics.HttpRequestDuration.Record(r.Context(), duration, attrs)
	})
}
