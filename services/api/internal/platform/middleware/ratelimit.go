package middleware

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// RateLimitByIP limits requests per client IP.
func RateLimitByIP(limit int, period time.Duration) gin.HandlerFunc {
	return rateLimit(limit, period, func(c *gin.Context) string { return "ip:" + c.ClientIP() })
}

// RateLimitByUser limits requests per authenticated user; it must run after
// RequireUser. Falls back to the client IP if no user is set.
func RateLimitByUser(limit int, period time.Duration) gin.HandlerFunc {
	return rateLimit(limit, period, func(c *gin.Context) string {
		if id := c.GetString("userID"); id != "" {
			return "user:" + id
		}
		return "ip:" + c.ClientIP()
	})
}

func rateLimit(limit int, period time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	limiter := ratelimit.New(limit, period)
	return func(c *gin.Context) {
		ok, retry := limiter.Allow(key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
