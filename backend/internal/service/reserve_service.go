package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/I000000/InventoryManagement/internal/metrics"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type reserveService struct {
	stockRepo StockRepository
	redis     *redis.Client
	logger    *zap.Logger
}

func NewReserveService(
	stockRepo StockRepository,
	redis *redis.Client,
	logger *zap.Logger,
) ReserveService {
	return &reserveService{
		stockRepo: stockRepo,
		redis:     redis,
		logger:    logger,
	}
}

func (s *reserveService) Reserve(ctx context.Context, req domain.ReserveRequest) (domain.ReserveResponse, error) {
	metrics.ActiveReservations.Inc()
	defer metrics.ActiveReservations.Dec()

	idempotencyKey := "idempotent:" + req.RequestID
	ok, err := s.redis.SetNX(ctx, idempotencyKey, "1", 5*time.Minute).Result()
	if err != nil {
		return domain.ReserveResponse{}, fmt.Errorf("redis setnx: %w", err)
	}
	if !ok {
		return domain.ReserveResponse{}, ErrDuplicateRequest
	}

	resp, err := s.stockRepo.ReserveTx(ctx, req)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrProductNotFound):
			s.redis.Del(ctx, "stock:"+req.ProductID)
			return domain.ReserveResponse{}, ErrProductNotFound
		case errors.Is(err, repository.ErrNotEnoughStock):
			s.redis.Del(ctx, "stock:"+req.ProductID)
			return domain.ReserveResponse{}, ErrNotEnoughStock
		case errors.Is(err, repository.ErrVersionConflict):
			return domain.ReserveResponse{}, ErrVersionConflict
		default:
			return domain.ReserveResponse{}, err
		}
	}

	metrics.ReservationsTotal.WithLabelValues(req.ProductID, "success").Inc()

	newAvailable, err := s.redis.DecrBy(ctx, "stock:"+req.ProductID, int64(req.Quantity)).Result()
	if err != nil {
		s.logger.Warn("failed to update stock cache", zap.String("product_id", req.ProductID), zap.Error(err))
	} else {
		s.redis.Expire(ctx, "stock:"+req.ProductID, 5*time.Minute)
		s.logger.Debug("stock cache updated", zap.String("product_id", req.ProductID), zap.Int64("new_available", newAvailable))
	}

	return resp, nil
}
