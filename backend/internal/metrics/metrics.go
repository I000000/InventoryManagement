package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP метрики
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// Бизнес-метрики
	ReservationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reservations_total",
			Help: "Total number of reservation attempts",
		},
		[]string{"product_id", "status"}, // status: success, duplicate, not_enough, error
	)

	ActiveReservations = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_reservations",
			Help: "Current number of active reservations (in-flight)",
		},
	)
)

// Middleware для сбора HTTP метрик
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			RequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(v)
		}))
		defer timer.ObserveDuration()

		c.Next()

		status := c.Writer.Status()
		RequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), string(rune(status))).Inc()
	}
}
