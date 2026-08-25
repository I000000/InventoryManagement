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
	defer tx.Rollback()

	var available int
	query := `SELECT total_count - reserved_count AS available 
              FROM stocks WHERE product_id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, query, req.ProductID).Scan(&available)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ReserveResponse{}, repository.ErrProductNotFound
		}
		return domain.ReserveResponse{}, fmt.Errorf("select for update: %w", err)
	}

	if available < req.Quantity {
		return domain.ReserveResponse{}, repository.ErrNotEnoughStock
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE stocks 
         SET reserved_count = reserved_count + $1, 
             version = version + 1,
             updated_at = NOW()
         WHERE product_id = $2`,
		req.Quantity, req.ProductID,
	)
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("update stocks: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("commit tx: %w", err)
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
