package main

import (
	"context"
	"database/sql"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/kafka"
	"github.com/I000000/InventoryManagement/internal/migrate"
	"github.com/I000000/InventoryManagement/internal/repository/clickhouse"
	"go.uber.org/zap"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func main() {
	cfg := config.Load()
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

	if err := migrate.ApplyClickHouseMigrations(chDB); err != nil {
		logger.Fatal("ClickHouse migrations failed", zap.Error(err))
	}
	logger.Info("ClickHouse migrations applied")

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
