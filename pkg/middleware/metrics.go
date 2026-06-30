package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vsayfb/gig-platform-core-service/pkg/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

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

		if route == "/metrics" {
			return
		}

		status := strconv.Itoa(rw.status)
		duration := time.Since(start).Seconds()

		metrics.HttpRequestsTotal.
			WithLabelValues(route, r.Method, status).
			Inc()

		metrics.HttpRequestDuration.
			WithLabelValues(route, r.Method, status).
			Observe(duration)
	})
}
