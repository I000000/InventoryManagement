package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/jmoiron/sqlx"
)

type stockRepository struct {
	db *sqlx.DB
}

// Проверка, что реализация соответствует интерфейсу
var _ service.StockRepository = (*stockRepository)(nil)

func NewStockRepository(db *sqlx.DB) service.StockRepository {
	return &stockRepository{db: db}
}

func (r *stockRepository) ReserveTx(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx,
		`UPDATE stocks 
		 SET reserved_count = reserved_count + $1, 
		     version = version + 1,
		     updated_at = NOW()
		 WHERE product_id = $2 AND total_count - reserved_count >= $1`,
		req.Quantity, req.ProductID,
	)
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("update: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		var exists bool
		err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM stocks WHERE product_id=$1)", req.ProductID).Scan(&exists)
		if err != nil {
			return domain.ReserveResponse{}, fmt.Errorf("check exists: %w", err)
		}
		if !exists {
			return domain.ReserveResponse{}, repository.ErrProductNotFound
		}
		return domain.ReserveResponse{}, repository.ErrNotEnoughStock
	}

	if err := tx.Commit(); err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("commit: %w", err)
	}

	return domain.ReserveResponse{
		Status:    "reserved",
		ProductID: req.ProductID,
		Reserved:  req.Quantity,
	}, nil
}

func (r *stockRepository) GetAvailable(ctx context.Context, productID string) (int, error) {
	var available int
	err := r.db.QueryRowContext(ctx,
		"SELECT total_count - reserved_count FROM stocks WHERE product_id = $1",
		productID).Scan(&available)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, repository.ErrProductNotFound
		}
		return 0, err
	}
	return available, nil
}
