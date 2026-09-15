package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-ID"

const ContextRequestID = "request_id"

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

func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(ContextRequestID); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
