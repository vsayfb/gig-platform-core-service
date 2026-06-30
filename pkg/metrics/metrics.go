package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "core_http_requests_total",
			Help: "Total HTTP requests handled, by route, method and status.",
		},
		[]string{"route", "method", "status"},
	)

	HttpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "core_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status"},
	)
)

func Register() {
	prometheus.MustRegister(
		HttpRequestsTotal,
		HttpRequestDuration,
	)
}
