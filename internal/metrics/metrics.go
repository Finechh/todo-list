package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HttpRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP request",
		},
		[]string{"method", "path"},
	)
	BackendErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "Backend_error_total",
			Help: "Total number of backend error",
		},
		[]string{"backend", "operation"},
	)
	BackendSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "Backend_success_total",
			Help: "Total number of backend successful",
		},
		[]string{"backend", "operation"},
	)

	HttpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_second",
			Help:    "Duration of HTTP request in second",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	BackendDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "Backend_duration_second",
			Help:    "Duration of backend request in second",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend", "operation"},
	)
	CacheMiss = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "Cache_miss",
			Help: "Total number of cache miss",
		},
		[]string{"backend", "cache miss"},
	)
	HttpErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "Http_error_total",
			Help: "Total error of http",
		},
		[]string{"method", "path"},
	)
)

func Register() {
	prometheus.MustRegister(HttpRequestTotal)
	prometheus.MustRegister(HttpDuration)
	prometheus.MustRegister(BackendDuration)
	prometheus.MustRegister(BackendErrorTotal)
	prometheus.MustRegister(BackendSuccessTotal)
	prometheus.MustRegister(HttpErrorTotal)
	prometheus.MustRegister(CacheMiss)
}

func IncHttpRequestTotal(method, path string) {
	HttpRequestTotal.WithLabelValues(method, path).Inc()
}

func IncHttpErrorTotal(method, path string) {
	HttpErrorTotal.WithLabelValues(method, path).Inc()
}

func IncBackendError(backend, operation string) {
	BackendErrorTotal.WithLabelValues(backend, operation).Inc()
}
func IncBackendSuccess(backend, operation string) {
	BackendSuccessTotal.WithLabelValues(backend, operation).Inc()
}
func ObserveBackendDuration(backend, operation string, d time.Duration) {
	BackendDuration.WithLabelValues(backend, operation).Observe(d.Seconds())
}
func ObserveHttpDuration(method, path string, d time.Duration) {
	HttpDuration.WithLabelValues(method, path).Observe(d.Seconds())
}
