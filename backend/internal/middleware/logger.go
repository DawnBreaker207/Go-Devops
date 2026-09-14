package middleware

import (
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// sensitiveQueryKeys are credentials that travel in URLs (realtime token,
// payment return signatures) and must not be logged (RT-01).
var sensitiveQueryKeys = map[string]bool{"token": true, "sig": true, "signature": true}

// redactQuery masks credential values of a raw query string.
func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[unparseable]"
	}
	for key := range values {
		if sensitiveQueryKeys[key] {
			values[key] = []string{"REDACTED"}
		}
	}
	return values.Encode()
}

// Logger logs each request with latency and status.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactQuery(c.Request.URL.RawQuery)

		c.Next()

		fields := []zap.Field{
			zap.String("request_id", GetRequestID(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		}

		switch {
		case c.Writer.Status() >= 500:
			logger.L().Error("http request", append(fields, zap.String("errors", c.Errors.String()))...)
		case c.Writer.Status() >= 400:
			logger.L().Warn("http request", fields...)
		default:
			logger.L().Info("http request", fields...)
		}
	}
}
