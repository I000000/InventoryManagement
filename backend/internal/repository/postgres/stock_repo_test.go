package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStockRepository_ReserveTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	outboxRepo := NewOutboxRepository(sqlxDB)
	repo := NewStockRepository(sqlxDB, outboxRepo)

	ctx := context.Background()

	tests := []struct {
		name         string
		req          domain.ReserveRequest
		setupMock    func()
		expectedResp domain.ReserveResponse
		expectedErr  error
	}{
		{
			name: "successful reservation",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  2,
				RequestID: "req-123",
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE stocks 
					SET reserved_count = reserved_count \+ \$1, 
					    version = version \+ 1, 
					    updated_at = NOW\(\) 
					WHERE product_id = \$2 AND total_count - reserved_count >= \$1`).
					WithArgs(2, "iphone_15").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(`INSERT INTO outbox \(event_id, event_type, payload, status, traceparent, tracestate, request_id\) VALUES \(\$1, \$2, \$3, 'pending', \$4, \$5, \$6\)`).
					WithArgs("req-123", "stock_reserved", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
			expectedResp: domain.ReserveResponse{
				Status:    "reserved",
				ProductID: "iphone_15",
				Reserved:  2,
			},
			expectedErr: nil,
		},
		{
			name: "product not found (rows affected 0, no exists)",
			req: domain.ReserveRequest{
				ProductID: "unknown",
				Quantity:  1,
				RequestID: "req-456",
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE stocks 
					SET reserved_count = reserved_count \+ \$1, 
					    version = version \+ 1, 
					    updated_at = NOW\(\) 
					WHERE product_id = \$2 AND total_count - reserved_count >= \$1`).
					WithArgs(1, "unknown").
					WillReturnResult(sqlmock.NewResult(0, 0))

				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM stocks WHERE product_id=\$1\)`).
					WithArgs("unknown").
					WillReturnRows(rows)

				mock.ExpectRollback()
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  repository.ErrProductNotFound,
		},
		{
			name: "not enough stock (rows affected 0, exists)",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  100,
				RequestID: "req-789",
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE stocks 
					SET reserved_count = reserved_count \+ \$1, 
					    version = version \+ 1, 
					    updated_at = NOW\(\) 
					WHERE product_id = \$2 AND total_count - reserved_count >= \$1`).
					WithArgs(100, "iphone_15").
					WillReturnResult(sqlmock.NewResult(0, 0))

				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM stocks WHERE product_id=\$1\)`).
					WithArgs("iphone_15").
					WillReturnRows(rows)

				mock.ExpectRollback()
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  repository.ErrNotEnoughStock,
		},
		{
			name: "update fails",
			req: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  1,
				RequestID: "req-101",
			},
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE stocks 
					SET reserved_count = reserved_count \+ \$1, 
					    version = version \+ 1, 
					    updated_at = NOW\(\) 
					WHERE product_id = \$2 AND total_count - reserved_count >= \$1`).
					WithArgs(1, "iphone_15").
					WillReturnError(errors.New("update failed"))

				mock.ExpectRollback()
			},
			expectedResp: domain.ReserveResponse{},
			expectedErr:  errors.New("update: update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			resp, err := repo.ReserveTx(ctx, tt.req)

			if tt.expectedErr != nil {
				assert.ErrorContains(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStockRepository_GetAvailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewStockRepository(sqlxDB, nil)

	ctx := context.Background()

	tests := []struct {
		name        string
		productID   string
		setupMock   func()
		expected    int
		expectedErr error
	}{
		{
			name:      "success",
			productID: "iphone_15",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"available"}).AddRow(10)
				mock.ExpectQuery(`SELECT total_count - reserved_count FROM stocks WHERE product_id = \$1`).
					WithArgs("iphone_15").
					WillReturnRows(rows)
			},
			expected:    10,
			expectedErr: nil,
		},
		{
			name:      "product not found",
			productID: "unknown",
			setupMock: func() {
				mock.ExpectQuery(`SELECT total_count - reserved_count FROM stocks WHERE product_id = \$1`).
					WithArgs("unknown").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    0,
			expectedErr: repository.ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			avail, err := repo.GetAvailable(ctx, tt.productID)

			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, avail)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
