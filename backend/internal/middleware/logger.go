package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Logger ghi lai moi request kem thoi gian xu ly va status.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

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
