package service

import (
	"context"
	"fmt"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type reserveService struct {
	stockRepo StockRepository
	redis     *redis.Client
	producer  EventProducer
	logger    *zap.Logger
}

func NewReserveService(
	stockRepo StockRepository,
	redis *redis.Client,
	producer EventProducer,
	logger *zap.Logger,
) ReserveService {
	return &reserveService{
		stockRepo: stockRepo,
		redis:     redis,
		producer:  producer,
		logger:    logger,
	}
}

func (s *reserveService) Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error) {
	// Идемпотентность
	idempotencyKey := "idempotent:" + req.RequestID
	ok, err := s.redis.SetNX(ctx, idempotencyKey, "1", 5*time.Minute).Result()
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("redis setnx: %w", err)
	}
	if !ok {
		return domain.ReserveResponse{}, ErrDuplicateRequest
	}

	// Вызов репозитория
	resp, err := s.stockRepo.ReserveTx(ctx, req)
	if err != nil {
		return domain.ReserveResponse{}, err
	}

	// Асинхронная отправка в Kafka
	event := domain.StockReservedEvent{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		RequestID: req.RequestID,
		Timestamp: time.Now().Unix(),
	}
	go func() {
		if err := s.producer.SendStockReservedEvent(context.Background(), event); err != nil {
			s.logger.Error("failed to send Kafka event",
				zap.String("product_id", req.ProductID),
				zap.Error(err),
			)
		}
	}()

	return resp, nil
}
