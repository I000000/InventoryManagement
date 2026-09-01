package migrate

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

func ApplyPostgresMigrations(db *sql.DB) error {
	goose.SetBaseFS(postgresMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations/postgres"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
