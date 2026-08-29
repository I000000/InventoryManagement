package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/I000000/InventoryManagement/internal/domain"
	"github.com/I000000/InventoryManagement/internal/service"
	"github.com/gin-gonic/gin"
)

type ReserveHandler struct {
	reserveService service.ReserveService
}

func NewReserveHandler(svc service.ReserveService) *ReserveHandler {
	return &ReserveHandler{reserveService: svc}
}

func (h *ReserveHandler) Reserve(c *gin.Context) {
	var req domain.ReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.reserveService.Reserve(c.Request.Context(), req)
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
