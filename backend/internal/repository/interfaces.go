package repository

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
)

//go:generate mockery --name ReserveLogRepository --output ../mocks --outpkg mocks --case underscore
type ReserveLogRepository interface {
	Insert(ctx context.Context, productID string, quantity int, requestID, userID, status string, errMsg *string) error
	GetRecent(ctx context.Context, limit int) ([]domain.ReserveLogEntry, error)
}
