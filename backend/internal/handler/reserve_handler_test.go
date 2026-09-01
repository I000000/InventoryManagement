package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/mocks"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestReserveHandler_Reserve(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zap.NewNop()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.ReserveService)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful reservation",
			requestBody: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  2,
				RequestID: "req-123",
			},
			setupMock: func(m *mocks.ReserveService) {
				m.On("Reserve", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{
						Status:    "reserved",
						ProductID: "iphone_15",
						Reserved:  2,
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":     "reserved",
				"product_id": "iphone_15",
				"reserved":   float64(2),
			},
		},
		{
			name: "invalid request (missing product_id)",
			requestBody: map[string]interface{}{
				"quantity":   1,
				"request_id": "req-456",
			},
			setupMock: func(m *mocks.ReserveService) {
				// не вызывается
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "product not found",
			requestBody: domain.ReserveRequest{
				ProductID: "unknown",
				Quantity:  1,
				RequestID: "req-456",
			},
			setupMock: func(m *mocks.ReserveService) {
				m.On("Reserve", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					service.ErrProductNotFound,
				)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "not enough stock",
			requestBody: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  100,
				RequestID: "req-789",
			},
			setupMock: func(m *mocks.ReserveService) {
				m.On("Reserve", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					service.ErrNotEnoughStock,
				)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "duplicate request",
			requestBody: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  1,
				RequestID: "dup-123",
			},
			setupMock: func(m *mocks.ReserveService) {
				m.On("Reserve", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					service.ErrDuplicateRequest,
				)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "internal server error",
			requestBody: domain.ReserveRequest{
				ProductID: "iphone_15",
				Quantity:  1,
				RequestID: "req-101",
			},
			setupMock: func(m *mocks.ReserveService) {
				m.On("Reserve", mock.Anything, mock.Anything).Return(
					domain.ReserveResponse{},
					errors.New("unexpected error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewReserveService(t)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			h := NewReserveHandler(mockSvc, logger)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/api/v1/reserve", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			h.Reserve(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp domain.ReserveResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody["product_id"], resp.ProductID)
				assert.Equal(t, tt.expectedBody["status"], resp.Status)
				assert.Equal(t, int(tt.expectedBody["reserved"].(float64)), resp.Reserved)
			}

			// Проверка ожиданий автоматическая через Cleanup, зарегистрированный в NewReserveService
		})
	}
}
