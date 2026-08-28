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

	// --- Logger ---

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

	// Настройка пула соединений ClickHouse
	chDB.SetMaxOpenConns(10)
	chDB.SetMaxIdleConns(5)
	chDB.SetConnMaxLifetime(30 * time.Minute)
	chDB.SetConnMaxIdleTime(5 * time.Minute)

	logger.Info("ClickHouse connected, migrations must be applied manually")

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

	logger.Info("Worker started, listening for messages...")
	consumer.ConsumeWithRetry(ctx, handler)

	logger.Info("Worker shutting down gracefully")
}
