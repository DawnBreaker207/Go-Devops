// Package middleware provides Gin middleware for auth, rate limiting and
// audit declaration.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Audit declares that a route is audited with the given action and resource
// type. It stashes WHO is acting before the handler runs (services write the
// success row inside their transaction) and logs the failure when the status
// comes back >= 400, outside any transaction — nothing was changed. Register
// it BEFORE RequireRoles so forbidden attempts on an authenticated route are
// logged too. A failed audit write only logs; it never fails the request.
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