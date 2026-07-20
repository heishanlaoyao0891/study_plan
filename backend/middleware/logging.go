package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type requestLog struct {
	Level     string `json:"level"`
	RequestID string `json:"request_id"`
	UserID    uint   `json:"user_id,omitempty"`
	Method    string `json:"method"`
	Route     string `json:"route"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
	Time      string `json:"time"`
}

func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()

		level := "info"
		if c.Writer.Status() >= http.StatusInternalServerError || len(c.Errors) > 0 {
			level = "error"
		}
		uid, _ := c.Get(CtxUserIDKey)
		entry := requestLog{
			Level:     level,
			RequestID: requestID,
			Method:    c.Request.Method,
			Route:     c.FullPath(),
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			LatencyMS: time.Since(start).Milliseconds(),
			Error:     c.Errors.String(),
			Time:      time.Now().Format(time.RFC3339),
		}
		if userID, ok := uid.(uint); ok {
			entry.UserID = userID
		}
		data, _ := json.Marshal(entry)
		log.Println(string(data))
	}
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}
