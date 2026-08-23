package service

import (
	"context"
	"fmt"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/redis/go-redis/v9"
)

type reserveService struct {
	stockRepo StockRepository
	redis     *redis.Client
	// Kafka producer добавить позже
}

// NewReserveService — конструктор, возвращает интерфейс
func NewReserveService(stockRepo StockRepository, redisClient *redis.Client) ReserveService {
	return &reserveService{
		stockRepo: stockRepo,
		redis:     redisClient,
	}
}

func (s *reserveService) Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error) {
	// 1. Проверка идемпотентности через Redis
	idempotencyKey := "idempotent:" + req.RequestID
	exists, err := s.redis.Exists(ctx, idempotencyKey).Result()
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("redis check failed: %w", err)
	}
	if exists > 0 {
		return domain.ReserveResponse{}, ErrDuplicateRequest
	}

	// 2. Вызов репозитория
	resp, err := s.stockRepo.ReserveTx(ctx, req)
	if err != nil {
		return domain.ReserveResponse{}, err
	}

	// 3. Сохраняем ключ идемпотентности
	if err := s.redis.Set(ctx, idempotencyKey, "1", 5*time.Minute).Err(); err != nil {
		// Логируем ошибку, но не возвращаем её (резерв уже сделан)
		// log.Warn("failed to set idempotency key", err)
	}

	// 4. Отправка события в Kafka (асинхронно, пока заглушка)
	// go s.publishEvent(ctx, domain.StockReservedEvent{...})

	return resp, nil
}
