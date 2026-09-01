package service

import (
	"context"
	"testing"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/mocks"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReserveService_Reserve(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	ctx := context.Background()

	tests := []struct {
		name           string
		req            domain.ReserveRequest
		setupMocks     func(*mocks.StockRepository)
		expectedResp   domain.ReserveResponse
		expectedErr    error
		checkRedisKeys bool
	}{
		{
			name: "successful reservation",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  2,
				RequestID: "req-123",
			},
			setupMocks: func(repo *mocks.StockRepository) {
				repo.On("ReserveTx", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{
						Status:    "reserved",
						ProductID: "iphone_15",
						Reserved:  2,
					},
					nil,
				)
			},
			expectedResp: domain.ReserveResponse{
				Status:    "reserved",
				ProductID: "iphone_15",
				Reserved:  2,
			},
			expectedErr:    nil,
			checkRedisKeys: true,
		},
		{
			name: "duplicate request",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  1,
				RequestID: "dup-123",
			},
			setupMocks: func(repo *mocks.StockRepository) {
				// не вызывается
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  ErrDuplicateRequest,
		},
		{
			name: "product not found",
			req: domain.ReserveRequest{
				ProductID: "unknown",
				Quantity:  1,
				RequestID: "req-456",
			},
			setupMocks: func(repo *mocks.StockRepository) {
				repo.On("ReserveTx", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					repository.ErrProductNotFound,
				)
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  ErrProductNotFound,
		},
		{
			name: "not enough stock",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  100,
				RequestID: "req-789",
			},
			setupMocks: func(repo *mocks.StockRepository) {
				repo.On("ReserveTx", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					repository.ErrNotEnoughStock,
				)
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  ErrNotEnoughStock,
		},
		{
			name: "version conflict",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  1,
				RequestID: "req-101",
			},
			setupMocks: func(repo *mocks.StockRepository) {
				repo.On("ReserveTx", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					repository.ErrVersionConflict,
				)
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  ErrVersionConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr.FlushAll()

			if tt.name == "duplicate request" {
				err := rdb.Set(ctx, "idempotent:dup-123", "1", 5*time.Minute).Err()
				require.NoError(t, err)
			}

			repo := mocks.NewStockRepository(t)
			tt.setupMocks(repo)

			svc := &reserveService{
				stockRepo: repo,
				redis:     rdb,
				logger:    logger,
			}

			resp, err := svc.Reserve(ctx, tt.req)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Проверка ожиданий автоматическая
		})
	}
}
