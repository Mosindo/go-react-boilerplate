package realtime

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	heartbeatInterval = 20 * time.Second
	// Streams are closed after the access-token lifetime; clients reconnect
	// with a fresh token, so a revoked session cannot keep listening.
	maxStreamDuration = 15 * time.Minute
)

// StreamHandler serves Server-Sent Events for the authenticated user.
// SSE is plain HTTP: it needs no extra dependency, crosses proxies and works
// in React Native through XMLHttpRequest streaming.
func StreamHandler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")
		if userID == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		sub := hub.Subscribe(userID)
		defer hub.Unsubscribe(sub)

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream; charset=utf-8")
		h.Set("Cache-Control", "no-store")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)

		heartbeat := time.NewTicker(heartbeatInterval)
		defer heartbeat.Stop()
		deadline := time.NewTimer(maxStreamDuration)
		defer deadline.Stop()

		write := func(chunk string) bool {
			if _, err := io.WriteString(c.Writer, chunk); err != nil {
				return false
			}
			c.Writer.Flush()
			return true
		}
		// Padding helps some clients/proxies start delivering immediately.
		if !write("retry: 3000\n: connected\n\n") {
			return
		}
		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				write("event: reconnect\ndata: {}\n\n")
				return
			case <-heartbeat.C:
				if !write(": ping\n\n") {
					return
				}
			case payload, ok := <-sub.C:
				if !ok {
					return
				}
				if !write("data: " + string(payload) + "\n\n") {
					return
				}
			}
		}
	}
}
