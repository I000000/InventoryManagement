package postgres

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/jmoiron/sqlx"
)

var _ repository.ReserveLogRepository = (*ReserveLogRepository)(nil)

type ReserveLogRepository struct {
	db *sqlx.DB
}

func NewReserveLogRepository(db *sqlx.DB) *ReserveLogRepository {
	return &ReserveLogRepository{db: db}
}

func (r *ReserveLogRepository) GetRecent(ctx context.Context, limit int) ([]domain.ReserveLogEntry, error) {
	query := `SELECT id, product_id, quantity, request_id, user_id, status, error_message, created_at
	          FROM reserve_log
	          ORDER BY created_at DESC
	          LIMIT $1`
	var entries []domain.ReserveLogEntry
	err := r.db.SelectContext(ctx, &entries, query, limit)
	return entries, err
}

func (r *ReserveLogRepository) Insert(ctx context.Context, productID string, quantity int, requestID, userID, status string, errMsg *string) error {
	query := `INSERT INTO reserve_log (product_id, quantity, request_id, user_id, status, error_message)
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, productID, quantity, requestID, userID, status, errMsg)
	return err
}
