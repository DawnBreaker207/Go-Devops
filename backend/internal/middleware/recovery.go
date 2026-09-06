package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// Recovery bat panic, ghi log stack va tra ve 500 dung dinh dang chung.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.L().Error("panic recovered",
					zap.String("request_id", GetRequestID(c)),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Body{
					Code:    apperrors.CodeInternal,
					Message: "internal server error",
				})
			}
		}()
		c.Next()
	}
}
