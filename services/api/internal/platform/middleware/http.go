package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type metricKey struct {
	Method string
	Path   string
	Status int
}

type metricValue struct {
	Count         int64
	LatencyTotal  int64
	LatencyMax    int64
	LastUpdatedAt time.Time
}

type metricsStore struct {
	mu   sync.Mutex
	data map[metricKey]metricValue
}

var httpMetrics = metricsStore{
	data: make(map[metricKey]metricValue),
}

func RequestMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		method := c.Request.Method

		key := metricKey{Method: method, Path: path, Status: status}
		httpMetrics.mu.Lock()
		current := httpMetrics.data[key]
		current.Count++
		current.LatencyTotal += latency
		if latency > current.LatencyMax {
			current.LatencyMax = latency
		}
		current.LastUpdatedAt = time.Now().UTC()
		httpMetrics.data[key] = current
		total := current.Count
		avg := int64(0)
		if current.Count > 0 {
			avg = current.LatencyTotal / current.Count
		}
		httpMetrics.mu.Unlock()

		if total%50 == 0 {
			log.Printf(
				`{"event":"http_metric","method":%q,"path":%q,"status":%d,"count":%d,"avg_latency_ms":%d,"max_latency_ms":%d}`,
				method,
				path,
				status,
				total,
				avg,
				current.LatencyMax,
			)
		}
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		userID := c.GetString("userID")

		log.Printf(
			`{"event":"http_request","request_id":%q,"method":%q,"path":%q,"status":%d,"latency_ms":%d,"client_ip":%q,"user_id":%q}`,
			c.GetString("request_id"),
			c.Request.Method,
			path,
			c.Writer.Status(),
			requestLatencyFromContext(c),
			c.ClientIP(),
			userID,
		)
	}
}

func RequestStart() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("request_started_at", time.Now())
		c.Next()
	}
}

func requestLatencyFromContext(c *gin.Context) int64 {
	startedAt, ok := c.Get("request_started_at")
	if !ok {
		return 0
	}
	start, ok := startedAt.(time.Time)
	if !ok {
		return 0
	}
	return time.Since(start).Milliseconds()
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

// BodyLimit caps request bodies. Paths for which skip returns true (uploads) are left alone
// and must enforce their own, larger limit.
func BodyLimit(maxBytes int64, skip func(*gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && (skip == nil || !skip(c)) {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
