package service

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
)

type ReserveService interface {
	Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
}
