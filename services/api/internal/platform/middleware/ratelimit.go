package middleware

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// KeyByIP rate-limits by client IP (honouring TRUSTED_PROXIES only).
func KeyByIP(c *gin.Context) string {
	return "ip:" + c.ClientIP()
}

// KeyByUser rate-limits by authenticated user, falling back to IP.
func KeyByUser(c *gin.Context) string {
	if id := c.GetString("userID"); id != "" {
		return "user:" + id
	}
	return KeyByIP(c)
}

// RateLimit rejects requests once the key's bucket is empty.
func RateLimit(limiter *ratelimit.Limiter, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow(c.FullPath() + "|" + key(c)) {
			c.Header("Retry-After", "60")
			httpx.Abort(c, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down and retry shortly")
			return
		}
		c.Next()
	}
}

// MaxBodyBytes caps request body size. Paths starting with one of
// skipPrefixes are left to a route-specific limit (e.g. photo uploads).
func MaxBodyBytes(limit int64, skipPrefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(c.Request.URL.Path, prefix) {
				c.Next()
				return
			}
		}
		if c.Request.ContentLength > limit {
			httpx.Abort(c, http.StatusRequestEntityTooLarge, "payload_too_large", "request body too large")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
