// Package middleware provides the Gin middleware.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Audit must be registered before RequireRoles so forbidden attempts are logged too.
// Services write success rows in their transaction; failures (>= 400) are logged here.
func Audit(db *gorm.DB, action, resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rec := audit.FromGin(c, audit.Record{
			ActorID:      CurrentUserID(c),
			ActorRole:    CurrentUserRole(c),
			Action:       action,
			ResourceType: resourceType,
		})
		c.Request = c.Request.WithContext(audit.Stash(c.Request.Context(), rec))

		c.Next()

		if c.Writer.Status() >= http.StatusBadRequest {
			rec.Outcome = audit.OutcomeFailure
			rec.ErrorMessage = audit.ErrorMessage(c)
			if id := c.Param("id"); id != "" {
				rec.ResourceID = id
			}
			if err := audit.In(c.Request.Context(), db, rec); err != nil {
				logger.L().Warn("audit failure row not written",
					logger.Err(err), logger.String("action", action))
			}
		}
	}
}
