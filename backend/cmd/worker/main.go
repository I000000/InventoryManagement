package main

import (
	"context"
	"log"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/I000000/InventoryManagement/internal/config"
)

func main() {
	cfg := config.Load()

	log.Println("Starting stock worker...")
	log.Printf("Configuration:")
	log.Printf("  - ClickHouse: %s", cfg.CHHost)
	log.Printf("  - ClickHouse DB: %s", cfg.CHDB)
	log.Printf("  - Kafka Brokers: %s", cfg.KafkaBrokers)
	log.Printf("  - Kafka Topic: %s", cfg.KafkaTopic)

	// --- Подключение к ClickHouse ---
	chConn, err := ch.Open(&ch.Options{
		Addr: []string{cfg.CHHost},
		Auth: ch.Auth{
			Database: cfg.CHDB,
			Username: cfg.CHUser,
			Password: cfg.CHPass,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Printf("ClickHouse connection failed: %v", err)
	} else {
		defer chConn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := chConn.Ping(ctx); err != nil {
			log.Printf("ClickHouse ping failed: %v", err)
		} else {
			log.Println("Connected to ClickHouse successfully")
		}
	}

	log.Println("Worker is running in simulation mode...")
	log.Println("Will process Kafka messages and write to ClickHouse")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Worker is alive and waiting for messages...")
	}
}
