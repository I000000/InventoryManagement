package main

import (
	"context"
	"database/sql"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/kafka"
	"github.com/I000000/InventoryManagement/internal/repository/clickhouse"
	"go.uber.org/zap"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func main() {
	cfg := config.Load()

	// --- Логгер ---
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting stock worker",
		zap.String("clickhouse", cfg.CHHost),
		zap.String("kafka", cfg.KafkaBrokers),
	)

	// --- ClickHouse ---
	chConnStr := fmt.Sprintf("clickhouse://%s:%s@%s/%s",
		cfg.CHUser, cfg.CHPass, cfg.CHHost, cfg.CHDB)
	chDB, err := sql.Open("clickhouse", chConnStr)
	if err != nil {
		logger.Fatal("ClickHouse open failed", zap.Error(err))
	}
	defer chDB.Close()

	chDB.SetMaxOpenConns(10)
	chDB.SetMaxIdleConns(5)
	chDB.SetConnMaxLifetime(30 * time.Minute)
	chDB.SetConnMaxIdleTime(5 * time.Minute)

	logger.Info("ClickHouse connected")

	chRepo := clickhouse.NewEventRepository(chDB)

	// --- Kafka consumer ---
	consumer, err := kafka.NewConsumer(
		[]string{cfg.KafkaBrokers},
		cfg.KafkaGroup,
		cfg.KafkaTopic,
		logger,
	)
	if err != nil {
		logger.Fatal("Kafka consumer creation failed", zap.Error(err))
	}
	defer consumer.Close()

	// --- Graceful shutdown ---
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	handler := func(ctx context.Context, event domain.StockReservedEvent) error {
		logger.Info("Received event from Kafka",
			zap.String("product_id", event.ProductID),
			zap.Int("quantity", event.Quantity),
			zap.String("request_id", event.RequestID),
		)
		return chRepo.InsertReservation(ctx, event)
	}

	// Запускаем потребление в горутине
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.ConsumeWithRetry(ctx, handler)
	}()

	logger.Info("Worker started, listening for messages...")

	// Ожидаем сигнал (контекст отменится при SIGINT/SIGTERM)
	<-ctx.Done()
	logger.Info("Received shutdown signal, stopping worker...")

	// Закрываем consumer, чтобы прервать ConsumeWithRetry
	if err := consumer.Close(); err != nil {
		logger.Error("Error closing consumer", zap.Error(err))
	}

	// Дожидаемся завершения горутины
	if err := <-errCh; err != nil && err != context.Canceled {
		logger.Error("Consumer error after shutdown", zap.Error(err))
	}

	logger.Info("Worker shut down gracefully")
}
