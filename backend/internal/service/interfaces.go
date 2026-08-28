package service

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
)

// ReserveService — бизнес-логика резервирования
type ReserveService interface {
	Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
}

// StockRepository — интерфейс для работы с остатками (определён здесь, т.к. используется сервисом)
type StockRepository interface {
	ReserveTx(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error)
	GetAvailable(ctx context.Context, productID string) (int, error)
}

// EventProducer — отправка событий (определён здесь, т.к. используется сервисом)
type EventProducer interface {
	SendStockReservedEvent(ctx context.Context, event domain.StockReservedEvent) error
	Close() error
}
