package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/handler"
	"github.com/I000000/InventoryManagement/internal/kafka"
	"github.com/I000000/InventoryManagement/internal/repository/postgres"
	"github.com/I000000/InventoryManagement/internal/service"
)

func main() {
	cfg := config.Load()

	// --- Логгер ---
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// --- PostgreSQL ---
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode)
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
	}
	defer db.Close()
	logger.Info("PostgreSQL connected")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		logger.Warn("Redis ping failed", zap.Error(err))
	} else {
		logger.Info("Redis connected")
	}

	// --- Kafka Producer ---
	var producer service.EventProducer
	if cfg.KafkaBrokers != "" {
		var err error
		producer, err = kafka.NewProducer(
			[]string{cfg.KafkaBrokers},
			cfg.KafkaTopic,
			logger,
		)
		if err != nil {
			logger.Error("Kafka producer creation failed", zap.Error(err))
		} else {
			logger.Info("Kafka producer created")
			defer func() {
				if closer, ok := producer.(interface{ Close() error }); ok {
					_ = closer.Close()
				}
			}()
		}
	} else {
		logger.Warn("Kafka brokers not set, producer disabled")
	}

	// --- DI ---
	stockRepo := postgres.NewStockRepository(db)
	reserveSvc := service.NewReserveService(stockRepo, rdb, producer, logger)
	reserveHandler := handler.NewReserveHandler(reserveSvc)

	// --- Gin ---
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "inventory-api",
		})
	})
	r.POST("/api/v1/reserve", reserveHandler.Reserve)

	// --- Запуск ---
	addr := ":" + cfg.ServerPort
	logger.Info(fmt.Sprintf("Starting API on %s", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
