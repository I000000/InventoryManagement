package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/clickhouse/*.sql
var chMigrations embed.FS

func ApplyClickHouseMigrations(db *sql.DB) error {
	goose.SetBaseFS(chMigrations)
	if err := goose.SetDialect("clickhouse"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations/clickhouse"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
