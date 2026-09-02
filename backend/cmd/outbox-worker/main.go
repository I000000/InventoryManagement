package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/I000000/InventoryManagement/internal/circuitbreaker"
	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/kafka"
	"github.com/I000000/InventoryManagement/internal/repository/postgres"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/I000000/InventoryManagement/internal/tracing"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	// --- Трассировка ---
	shutdown, err := tracing.InitTracer("outbox-worker")
	if err != nil {
		logger.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	logger.Info("Starting outbox worker")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode)
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
	}
	defer db.Close()

	producer, err := kafka.NewProducer([]string{cfg.KafkaBrokers}, cfg.KafkaTopic, logger)
	if err != nil {
		logger.Fatal("Kafka producer creation failed", zap.Error(err))
	}
	defer producer.Close()

	outboxRepo := postgres.NewOutboxRepository(db)

	// --- Circuit Breaker ---
	cb := circuitbreaker.NewKafkaCircuitBreaker(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logger.Info("Outbox worker started, polling every 2s")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Outbox worker shutting down")
			return
		case <-ticker.C:
			processOutbox(ctx, outboxRepo, producer, logger, cb)
		}
	}
}

func processOutbox(ctx context.Context, outboxRepo *postgres.OutboxRepository, producer service.EventProducer, logger *zap.Logger, cb *gobreaker.CircuitBreaker) {
	events, err := outboxRepo.GetPending(ctx, 10)
	if err != nil {
		logger.Error("failed to get pending events", zap.Error(err))
		return
	}
	if len(events) == 0 {
		return
	}

	for _, event := range events {
		if err := processEvent(ctx, event, producer, logger, cb); err != nil {
			logger.Error("failed to process event", zap.Int64("id", event.ID), zap.Error(err))
			if markErr := outboxRepo.MarkFailed(ctx, event.ID, err.Error()); markErr != nil {
				logger.Error("failed to mark event as failed", zap.Int64("id", event.ID), zap.Error(markErr))
			}
		} else {
			if markErr := outboxRepo.MarkProcessed(ctx, event.ID); markErr != nil {
				logger.Error("failed to mark event as processed", zap.Int64("id", event.ID), zap.Error(markErr))
			}
		}
	}
}

func processEvent(ctx context.Context, event postgres.OutboxEvent, producer service.EventProducer, logger *zap.Logger, cb *gobreaker.CircuitBreaker) error {
	if event.EventType != "stock_reserved" {
		logger.Warn("unknown event type", zap.String("event_type", event.EventType))
		return nil
	}

	// Извлекаем контекст из сохранённых заголовков
	carrier := propagation.MapCarrier{
		"traceparent": event.TraceParent,
		"tracestate":  event.TraceState,
	}
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, carrier)

	var stockEvent domain.StockReservedEvent
	if err := json.Unmarshal(event.Payload, &stockEvent); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// Создаём спан для отправки в Kafka
	ctx, span := tracing.TraceKafkaProduce(ctx, "stock-events", stockEvent.RequestID)
	defer span.End()

	// Отправляем через Circuit Breaker
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, producer.SendStockReservedEvent(ctx, stockEvent)
	})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("send to Kafka (circuit breaker): %w", err)
	}

	logger.Debug("event sent", zap.Int64("id", event.ID))
	return nil
}
