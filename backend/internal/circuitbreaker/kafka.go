package circuitbreaker

import (
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// NewKafkaCircuitBreaker создаёт Circuit Breaker для отправки в Kafka
func NewKafkaCircuitBreaker(logger *zap.Logger) *gobreaker.CircuitBreaker {
	logger = logger.With(zap.String("service", "circuit-breaker"))

	settings := gobreaker.Settings{
		Name:        "KafkaProducer",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		IsSuccessful: func(err error) bool {
			return err == nil
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit Breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}
	return gobreaker.NewCircuitBreaker(settings)
}
