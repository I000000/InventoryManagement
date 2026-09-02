package service

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
)

// ReserveService — бизнес-логика резервирования
//
//go:generate mockery --name ReserveService --output ../mocks --outpkg mocks --case underscore
type ReserveService interface {
	Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
}

// StockRepository — интерфейс для работы с остатками (определён здесь, т.к. используется сервисом)
//
//go:generate mockery --name StockRepository --output ../mocks --outpkg mocks --case underscore
type StockRepository interface {
	ReserveTx(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
	GetAvailable(ctx context.Context, productID string) (int, error)
}

// EventProducer — отправка событий (определён здесь, т.к. используется сервисом)
//
//go:generate mockery --name EventProducer --output ../mocks --outpkg mocks --case underscore
type EventProducer interface {
	SendStockReservedEvent(ctx context.Context, event domain.StockReservedEvent) error
	Close() error
}

// ReserveLogRepository — интерфейс для работы с логами резервирований (используется в хендлере)
//
//go:generate mockery --name ReserveLogRepository --output ../mocks --outpkg mocks --case underscore
type ReserveLogRepository interface {
	Insert(ctx context.Context, productID string, quantity int, requestID, userID, status string, errMsg *string) error
	GetRecent(ctx context.Context, limit int) ([]any, error) // или используй конкретный тип из postgres
}
