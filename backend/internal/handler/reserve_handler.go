package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/I000000/InventoryManagement/internal/domain"
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
	var req domain.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
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
			h.logger.Error("failed to insert reserve log", zap.Error(err))
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
				zap.String("product_id", req.ProductID),
				zap.String("status", status),
			)
		}
	}()

	// Возвращаем ответ
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrNotEnoughStock):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrDuplicateRequest):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrVersionConflict):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ReserveHandler) GetReservations(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	entries, err := h.reserveLogRepo.GetRecent(c.Request.Context(), limit)
	if err != nil {
		h.logger.Error("failed to get reserve log", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get reservations"})
		return
	}
	c.JSON(http.StatusOK, entries)
}
