package clickhouse

import (
	"context"
	"database/sql"

	"github.com/I000000/InventoryManagement/internal/domain"
)

type EventRepository interface {
	InsertReservation(ctx context.Context, event domain.StockReservedEvent) error
}

type clickhouseRepo struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &clickhouseRepo{db: db}
}

func (r *clickhouseRepo) InsertReservation(ctx context.Context, event domain.StockReservedEvent) error {
	query := `INSERT INTO stock_reservations (product_id, quantity, user_id, request_id) 
              VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, event.ProductID, event.Quantity, "", event.RequestID)
	return err
}
