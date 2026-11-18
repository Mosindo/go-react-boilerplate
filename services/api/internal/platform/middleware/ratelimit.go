package middleware

import (
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// KeyFunc extracts the rate limit key for a request.
type KeyFunc func(c *gin.Context) string

// ByClientIP keys anonymous endpoints (register, login, password reset).
func ByClientIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// ByUser keys authenticated endpoints; it must run after RequireUser.
func ByUser(c *gin.Context) string {
	if userID := c.GetString("userID"); userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.ClientIP()
}

// RateLimit rejects requests with 429 once the bucket for the key is empty.
func RateLimit(limiter *ratelimit.Limiter, scope string, key KeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := limiter.Allow(scope + "|" + key(c))
		if !ok {
			seconds := int(wait / time.Second)
			if wait%time.Second != 0 {
				seconds++
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please slow down",
				"code":  "rate_limited",
			})
			return
		}
		c.Next()
	}
}
