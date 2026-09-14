package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/database"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// RateLimit rejects requests over the per-IP frequency cap with 429.
// Place after Auth to rate-limit by user instead of IP.
func RateLimit(lim *ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !lim.Allow(c.ClientIP()) {
			response.Abort(c, apperrors.TooManyRequests("too many requests, try again later"))
			return
		}
		c.Next()
	}
}

// DBGuard checks the DB is alive before each request and answers 503 fast
// instead of letting every request hang or fail one by one.
func DBGuard(db *gorm.DB, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := database.Healthy(db, timeout); err != nil {
			response.Abort(c, apperrors.ServiceUnavailable("service busy, try again later"))
			return
		}
		c.Next()
	}
}