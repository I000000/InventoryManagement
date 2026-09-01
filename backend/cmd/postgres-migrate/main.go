package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/migrate"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	if err := migrate.ApplyPostgresMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("PostgreSQL migrations applied successfully")
}
