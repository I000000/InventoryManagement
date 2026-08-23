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

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/handler"
	"github.com/I000000/InventoryManagement/internal/repository/postgres"
	"github.com/I000000/InventoryManagement/internal/service"
)

func main() {
	cfg := config.Load()

	// --- PostgreSQL ---
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode)
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ PostgreSQL: %v", err)
	}
	defer db.Close()
	log.Println("✅ PostgreSQL connected")

	// --- Redis ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Printf("⚠️ Redis ping: %v", err)
	} else {
		log.Println("✅ Redis connected")
	}

	// --- Сборка зависимостей (DI) ---
	stockRepo := postgres.NewStockRepository(db)
	reserveSvc := service.NewReserveService(stockRepo, rdb)
	reserveHandler := handler.NewReserveHandler(reserveSvc)

	// --- Gin ---
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/api/v1/reserve", reserveHandler.Reserve)

	log.Printf("🚀 Starting API on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("❌ Server: %v", err)
	}
}
