package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/I000000/InventoryManagement/internal/metrics"
	"github.com/I000000/InventoryManagement/internal/middleware"
	"github.com/I000000/InventoryManagement/internal/tracing"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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
	defer func() { _ = logger.Sync() }()

	// --- Трассировка ---
	shutdown, err := tracing.InitTracer("inventory-api")
	if err != nil {
		logger.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer", zap.Error(err))
		}
	}()

	// --- PostgreSQL ---
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode)
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.DBMaxConns)
	db.SetMaxIdleConns(cfg.DBMaxConns / 2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	logger.Info("PostgreSQL connected with connection pool")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

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
			logger.Warn("Kafka producer creation failed, using no-op producer", zap.Error(err))
			producer = kafka.NewNoopProducer()
		} else {
			logger.Info("Kafka producer created")
		}
	} else {
		logger.Warn("Kafka brokers not set, using no-op producer")
		producer = kafka.NewNoopProducer()
	}
	defer func() {
		if err := producer.Close(); err != nil {
			logger.Error("failed to close Kafka producer", zap.Error(err))
		}
	}()

	// --- DI ---
	outboxRepo := postgres.NewOutboxRepository(db)
	stockRepo := postgres.NewStockRepository(db, outboxRepo)
	reserveSvc := service.NewReserveService(stockRepo, rdb, logger)
	reserveHandler := handler.NewReserveHandler(reserveSvc, logger)

	// --- Gin ---
	r := gin.Default()

	// Трассировка для HTTP запросов
	r.Use(otelgin.Middleware("inventory-api"))
	r.Use(metrics.MetricsMiddleware())
	r.Use(middleware.RateLimiterMiddleware(rdb, cfg.RateLimit, cfg.RateLimitWindow))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "inventory-api",
		})
	})
	r.POST("/api/v1/reserve", reserveHandler.Reserve)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		logger.Info("Starting API server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down API server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("API server exited")
}
