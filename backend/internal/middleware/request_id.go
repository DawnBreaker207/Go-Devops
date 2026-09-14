package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID is the request identifier header.
const HeaderRequestID = "X-Request-ID"

// ContextRequestID is the key storing the request id in the gin context.
const ContextRequestID = "request_id"

// RequestID attaches an id to each request for log tracing.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set(ContextRequestID, requestID)
		c.Writer.Header().Set(HeaderRequestID, requestID)
		c.Next()
	}
}

// GetRequestID returns the request id from context.
func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(ContextRequestID); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
