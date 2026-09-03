package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/middleware"
	"github.com/I000000/InventoryManagement/internal/repository"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/I000000/InventoryManagement/internal/websocket"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReserveHandler struct {
	reserveService service.ReserveService
	reserveLogRepo repository.ReserveLogRepository
	logger         *zap.Logger
	hub            *websocket.Hub
}

func NewReserveHandler(
	svc service.ReserveService,
	reserveLogRepo repository.ReserveLogRepository,
	logger *zap.Logger,
	hub *websocket.Hub,
) *ReserveHandler {
	return &ReserveHandler{
		reserveService: svc,
		reserveLogRepo: reserveLogRepo,
		logger:         logger,
		hub:            hub,
	}
}

func (h *ReserveHandler) Reserve(c *gin.Context) {
	requestID := middleware.GetRequestIDFromGin(c)
	ctx := context.WithValue(c.Request.Context(), middleware.RequestIDKey, requestID)

	var req domain.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Debug("reserve request",
		zap.String("request_id", requestID),
		zap.String("product_id", req.ProductID),
		zap.Int("quantity", req.Quantity),
	)

	resp, err := h.reserveService.Reserve(ctx, req)

	// Определяем статус и сообщение для лога
	var status string
	var errMsg *string
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			status = "not_found"
		case errors.Is(err, service.ErrNotEnoughStock):
			status = "not_enough_stock"
		case errors.Is(err, service.ErrDuplicateRequest):
			status = "duplicate"
		case errors.Is(err, service.ErrVersionConflict):
			status = "version_conflict"
		case errors.Is(err, context.DeadlineExceeded):
			status = "timeout"
		default:
			status = "error"
		}
		msg := err.Error()
		errMsg = &msg
	} else {
		status = "success"
	}

	// Асинхронная запись в лог и отправка WebSocket события
	go func() {
		logCtx := context.Background()
		if err := h.reserveLogRepo.Insert(logCtx, req.ProductID, req.Quantity, req.RequestID, "", status, errMsg); err != nil {
			h.logger.Error("failed to insert reserve log",
				zap.String("request_id", requestID),
				zap.Error(err),
			)
		} else {
			// Отправляем событие всем клиентам
			h.hub.Broadcast(websocket.Message{
				Type: "new_reservation",
				Payload: map[string]interface{}{
					"product_id": req.ProductID,
					"quantity":   req.Quantity,
					"status":     status,
					"created_at": time.Now().UTC().Format(time.RFC3339),
				},
			})
			h.logger.Debug("Broadcasting new reservation via WebSocket",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
				zap.String("status", status),
			)
		}
	}()

	// Возвращаем ответ
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			h.logger.Info("product not found",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrNotEnoughStock):
			h.logger.Info("not enough stock",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
				zap.Int("quantity", req.Quantity),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrDuplicateRequest):
			h.logger.Info("duplicate request",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrVersionConflict):
			h.logger.Info("version conflict",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, context.DeadlineExceeded):
			h.logger.Warn("request timeout",
				zap.String("request_id", requestID),
				zap.String("product_id", req.ProductID),
			)
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})

		default:
			h.logger.Error("unexpected error",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("product_id", req.ProductID),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	// Успешный ответ
	h.logger.Debug("reservation successful",
		zap.String("request_id", requestID),
		zap.String("product_id", req.ProductID),
		zap.Int("reserved", resp.Reserved),
	)
	c.JSON(http.StatusOK, resp)
}

func (h *ReserveHandler) GetReservations(c *gin.Context) {
	requestID := middleware.GetRequestIDFromGin(c)

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	entries, err := h.reserveLogRepo.GetRecent(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("failed to get reserve log",
			zap.String("request_id", requestID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get reservations"})
		return
	}
	c.JSON(http.StatusOK, entries)
}
