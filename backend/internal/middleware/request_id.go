package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HeaderRequestID la header mang ma dinh danh request.
const HeaderRequestID = "X-Request-ID"

// ContextRequestID la key luu request id trong gin.Context.
const ContextRequestID = "request_id"

// RequestID gan request id cho moi request de trace log xuyen suot.
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

// GetRequestID lay request id tu context.
func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(ContextRequestID); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
