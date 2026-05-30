package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed by the API.",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"method", "endpoint", "status"},
	)

	activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_requests",
			Help: "Current number of active HTTP requests.",
		},
	)

	tournamentRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tournament_requests_total",
			Help: "Total number of tournament-related HTTP requests processed by the API.",
		},
		[]string{"method", "endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDurationSeconds)
	prometheus.MustRegister(activeRequests)
	prometheus.MustRegister(tournamentRequestsTotal)
}

// MetricsHandler exposes metrics in Prometheus text format.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// Middleware records basic API metrics for every HTTP request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		activeRequests.Inc()
		defer activeRequests.Dec()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		endpoint := normalizeEndpoint(r.URL.Path)
		status := strconv.Itoa(recorder.statusCode)
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(r.Method, endpoint, status).Observe(duration)

		if isTournamentEndpoint(endpoint) {
			tournamentRequestsTotal.WithLabelValues(r.Method, endpoint, status).Inc()
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// normalizeEndpoint prevents high-cardinality labels.
// For the educational project it keeps the route readable and masks numeric ids.
func normalizeEndpoint(path string) string {
	if path == "" {
		return "/"
	}

	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}

		if isNumeric(part) {
			parts[i] = ":id"
		}
	}

	return strings.Join(parts, "/")
}

func isNumeric(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return value != ""
}

func isTournamentEndpoint(endpoint string) bool {
	return strings.Contains(strings.ToLower(endpoint), "tournament")
}
