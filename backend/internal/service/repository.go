package service

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
)

type StockRepository interface {
	ReserveTx(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
	GetAvailable(ctx context.Context, productID string) (int, error)
}
