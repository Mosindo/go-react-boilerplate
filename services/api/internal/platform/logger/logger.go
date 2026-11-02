package logger

import (
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var sensitiveValuePatterns = []struct {
	re          *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-\._~\+/]+=*`), "Bearer [REDACTED]"},
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`), "[REDACTED_JWT]"},
	{regexp.MustCompile(`(?i)(refresh[_-]?token|password|secret|authorization)(["'=:\s]+)([^",\s]+)`), `${1}${2}[REDACTED]`},
}

func LogHandlerError(c *gin.Context, operation string, status int, err error) {
	requestID := c.GetString("request_id")
	userID := c.GetString("userID")
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	log.Printf(
		`{"event":"handler_error","request_id":%q,"operation":%q,"method":%q,"path":%q,"user_id":%q,"status":%d,"latency_ms":%d,"error":%q}`,
		requestID,
		operation,
		c.Request.Method,
		path,
		userID,
		status,
		requestLatencyMs(c),
		sanitizeLogValue(err.Error()),
	)
}

func LogHandlerEvent(c *gin.Context, operation string, status int, fields map[string]string) {
	requestID := c.GetString("request_id")
	userID := c.GetString("userID")
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	details := ""
	for _, key := range keys {
		details += `,"` + sanitizeLogValue(key) + `":` + `"` + sanitizeLogValue(fields[key]) + `"`
	}
	log.Printf(
		`{"event":"handler_event","request_id":%q,"operation":%q,"method":%q,"path":%q,"user_id":%q,"status":%d,"latency_ms":%d%s}`,
		requestID,
		operation,
		c.Request.Method,
		path,
		userID,
		status,
		requestLatencyMs(c),
		details,
	)
}

func requestLatencyMs(c *gin.Context) int64 {
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

func sanitizeLogValue(v string) string {
	trimmed := strings.TrimSpace(v)
	trimmed = strings.ReplaceAll(trimmed, `\`, `\\`)
	trimmed = strings.ReplaceAll(trimmed, `"`, `\"`)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.ReplaceAll(trimmed, "\r", " ")
	for _, pattern := range sensitiveValuePatterns {
		trimmed = pattern.re.ReplaceAllString(trimmed, pattern.replacement)
	}
	if len(trimmed) > 256 {
		return trimmed[:256]
	}
	return trimmed
}
