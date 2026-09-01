package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/I000000/InventoryManagement/internal/config"
	"github.com/I000000/InventoryManagement/internal/migrate"
)

func main() {
	cfg := config.Load()

	connStr := fmt.Sprintf("clickhouse://%s:%s@%s/%s",
		cfg.CHUser, cfg.CHPass, cfg.CHHost, cfg.CHDB)
	db, err := sql.Open("clickhouse", connStr)
	if err != nil {
		log.Fatalf("Failed to open ClickHouse: %v", err)
	}
	defer db.Close()

	if err := migrate.ApplyClickHouseMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("ClickHouse migrations applied successfully")
}
