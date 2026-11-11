package middleware

import (
	"math"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// KeyByIP limits per client IP (respecting the configured trusted proxies).
func KeyByIP(c *gin.Context) string { return "ip:" + c.ClientIP() }

// KeyByUser limits per authenticated user, falling back to IP.
func KeyByUser(c *gin.Context) string {
	if id := httpx.UserID(c); id != "" {
		return "user:" + id
	}
	return KeyByIP(c)
}

// RateLimit rejects requests with 429 once the limiter bucket for the key is empty.
func RateLimit(l *ratelimit.Limiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, wait := l.Allow(c.FullPath() + "|" + key(c))
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			httpx.Fail(c, http.StatusTooManyRequests, "rate_limited", "Trop de requêtes. Réessaie dans un instant.", nil)
			return
		}
		c.Next()
	}
}
