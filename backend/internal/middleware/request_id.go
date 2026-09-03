package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestIDMiddleware генерирует или извлекает request_id из заголовка X-Request-ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(string(RequestIDKey), requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestIDFromContext извлекает request_id из контекста
func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestIDFromGin извлекает request_id из gin.Context
func GetRequestIDFromGin(c *gin.Context) string {
	if id, ok := c.Get(string(RequestIDKey)); ok {
		return id.(string)
	}
	return ""
}

// ContextWithRequestID возвращает новый контекст с request_id
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
