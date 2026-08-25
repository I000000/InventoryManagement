package service

import (
	"context"
	"errors"
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
	// 1. Идемпотентность
	idempotencyKey := "idempotent:" + req.RequestID
	exists, err := s.redis.Exists(ctx, idempotencyKey).Result()
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("redis check: %w", err)
	}
	if exists > 0 {
		return domain.ReserveResponse{}, errors.New("duplicate request")
	}

	// 2. Вызов репозитория
	resp, err := s.stockRepo.ReserveTx(ctx, req)
	if err != nil {
		return domain.ReserveResponse{}, err
	}

	// 3. Сохраняем ключ идемпотентности
	s.redis.Set(ctx, idempotencyKey, "1", 5*time.Minute)

	// 4. Асинхронная отправка в Kafka
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
