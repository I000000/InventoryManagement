package kafka

import (
	"context"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/service"
)

type noopProducer struct{}

func NewNoopProducer() service.EventProducer {
	return &noopProducer{}
}

func (p *noopProducer) SendStockReservedEvent(ctx context.Context, event domain.StockReservedEvent) error {
	return nil
}

func (p *noopProducer) Close() error {
	return nil
}
