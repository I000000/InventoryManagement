package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

type ReserveLogEntry struct {
	ID           int64     `db:"id" json:"id"`
	ProductID    string    `db:"product_id" json:"product_id"`
	Quantity     int       `db:"quantity" json:"quantity"`
	RequestID    string    `db:"request_id" json:"request_id"`
	UserID       string    `db:"user_id" json:"user_id"`
	Status       string    `db:"status" json:"status"`
	ErrorMessage *string   `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type ReserveLogRepository struct {
	db *sqlx.DB
}

func NewReserveLogRepository(db *sqlx.DB) *ReserveLogRepository {
	return &ReserveLogRepository{db: db}
}

func (r *ReserveLogRepository) GetRecent(ctx context.Context, limit int) ([]ReserveLogEntry, error) {
	query := `SELECT id, product_id, quantity, request_id, user_id, status, error_message, created_at
	          FROM reserve_log
	          ORDER BY created_at DESC
	          LIMIT $1`
	var entries []ReserveLogEntry
	err := r.db.SelectContext(ctx, &entries, query, limit)
	return entries, err
}

func (r *ReserveLogRepository) Insert(ctx context.Context, productID string, quantity int, requestID, userID, status string, errMsg *string) error {
	query := `INSERT INTO reserve_log (product_id, quantity, request_id, user_id, status, error_message)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, productID, quantity, requestID, userID, status, errMsg)
	return err
}
