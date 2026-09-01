package middleware

import (
	"net/http"
	"time"

	"github.com/I000000/InventoryManagement/internal/metrics"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

// RateLimiterMiddleware — middleware для ограничения запросов
func RateLimiterMiddleware(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	limiter := redis_rate.NewLimiter(rdb)

	return func(c *gin.Context) {
		// Используем IP клиента как ключ для лимитера
		key := c.ClientIP()

		res, err := limiter.Allow(c.Request.Context(), key, redis_rate.Limit{
			Rate:   limit,  // количество запросов
			Burst:  limit,  // можно разрешить кратковременный всплеск
			Period: window, // за временной период (окно)
		})
		if err != nil {
			// Если Redis недоступен, пропускаем запрос
			c.Next()
			return
		}

		if res.Allowed == 0 {
			metrics.RateLimitedRequests.Inc()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
