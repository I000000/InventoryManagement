package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/kafka"
	"github.com/I000000/InventoryManagement/internal/repository/postgres"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

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
			processOutbox(ctx, outboxRepo, producer, logger)
		}
	}
}

func processOutbox(ctx context.Context, outboxRepo *postgres.OutboxRepository, producer service.EventProducer, logger *zap.Logger) {
	events, err := outboxRepo.GetPending(ctx, 10)
	if err != nil {
		logger.Error("failed to get pending events", zap.Error(err))
		return
	}
	if len(events) == 0 {
		return
	}

	logger.Info("found pending events", zap.Int("count", len(events)))

	for _, event := range events {
		logger.Info("processing event", zap.Int64("id", event.ID), zap.String("event_type", event.EventType))
		if err := processEvent(ctx, event, producer, logger); err != nil {
			logger.Error("failed to process event", zap.Int64("id", event.ID), zap.Error(err))
			if err2 := outboxRepo.MarkFailed(ctx, event.ID, err.Error()); err2 != nil {
				logger.Error("failed to mark event as failed", zap.Int64("id", event.ID), zap.Error(err2))
			}
		} else {
			logger.Info("event processed successfully", zap.Int64("id", event.ID))
			if err2 := outboxRepo.MarkProcessed(ctx, event.ID); err2 != nil {
				logger.Error("failed to mark event as processed", zap.Int64("id", event.ID), zap.Error(err2))
			}
		}
	}
}

func processEvent(ctx context.Context, event postgres.OutboxEvent, producer service.EventProducer, logger *zap.Logger) error {
	if event.EventType != "stock_reserved" {
		logger.Warn("unknown event type", zap.String("event_type", event.EventType))
		return nil
	}
	var stockEvent domain.StockReservedEvent
	if err := json.Unmarshal(event.Payload, &stockEvent); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	logger.Info("sending event to Kafka", zap.Int64("id", event.ID), zap.String("product_id", stockEvent.ProductID))
	if err := producer.SendStockReservedEvent(ctx, stockEvent); err != nil {
		return fmt.Errorf("send to Kafka: %w", err)
	}
	logger.Info("event sent to Kafka", zap.Int64("id", event.ID))
	return nil
}
