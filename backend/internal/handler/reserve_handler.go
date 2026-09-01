package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ReserveHandler struct {
	reserveService service.ReserveService
	logger         *zap.Logger
}

func NewReserveHandler(svc service.ReserveService, logger *zap.Logger) *ReserveHandler {
	return &ReserveHandler{
		reserveService: svc,
		logger:         logger,
	}
}

func (h *ReserveHandler) Reserve(c *gin.Context) {
	var req domain.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Логируем входящий запрос (можно на DEBUG уровне)
	h.logger.Debug("reserve request",
		zap.String("product_id", req.ProductID),
		zap.Int("quantity", req.Quantity),
		zap.String("request_id", req.RequestID),
	)

	resp, err := h.reserveService.Reserve(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProductNotFound):
			h.logger.Info("product not found",
				zap.String("product_id", req.ProductID),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrNotEnoughStock):
			h.logger.Info("not enough stock",
				zap.String("product_id", req.ProductID),
				zap.Int("quantity", req.Quantity),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrDuplicateRequest):
			h.logger.Info("duplicate request",
				zap.String("product_id", req.ProductID),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, service.ErrVersionConflict):
			h.logger.Info("version conflict",
				zap.String("product_id", req.ProductID),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		case errors.Is(err, context.DeadlineExceeded):
			h.logger.Warn("request timeout",
				zap.String("product_id", req.ProductID),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "request timeout"})

		default:
			h.logger.Error("unexpected error",
				zap.Error(err),
				zap.String("product_id", req.ProductID),
				zap.String("request_id", req.RequestID),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	// Успешный ответ
	h.logger.Debug("reservation successful",
		zap.String("product_id", req.ProductID),
		zap.Int("reserved", resp.Reserved),
		zap.String("request_id", req.RequestID),
	)
	c.JSON(http.StatusOK, resp)
}
