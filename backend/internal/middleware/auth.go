package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

const (
	ContextUserID    = "user_id"
	ContextUserEmail = "user_email"
	ContextUserRole  = "user_role"
)

// AccountChecker lets a lock or role change take effect before the access token expires.
type AccountChecker interface {
	Status(ctx context.Context, userID string) (active bool, role string, err error)
}

func Auth(jwtManager *jwt.Manager, accounts AccountChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
			response.Abort(c, apperrors.Unauthorized("authorization header is required"))
			return
		}
		if authenticate(c, jwtManager, accounts) {
			c.Next()
		}
	}
}

// OptionalAuth lets anonymous through, but a sent token must be valid (401, not silent guest).
func OptionalAuth(jwtManager *jwt.Manager, accounts AccountChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" || authenticate(c, jwtManager, accounts) {
			c.Next()
		}
	}
}

func authenticate(c *gin.Context, jwtManager *jwt.Manager, accounts AccountChecker) bool {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		response.Abort(c, apperrors.Unauthorized("authorization header must be in format: Bearer <token>"))
		return false
	}

	claims, err := jwtManager.ParseAccess(parts[1])
	if err != nil {
		response.Abort(c, err)
		return false
	}
	if accounts != nil {
		active, role, err := accounts.Status(c.Request.Context(), claims.UserID)
		switch {
		case err != nil:
			response.Abort(c, err)
			return false
		case !active:
			response.Abort(c, apperrors.ErrAccountLocked)
			return false
		case role != claims.Role:
			// A new role needs a new token: log in again.
			response.Abort(c, apperrors.ErrInvalidToken)
			return false
		}
	}

	c.Set(ContextUserID, claims.UserID)
	c.Set(ContextUserEmail, claims.Email)
	c.Set(ContextUserRole, claims.Role)
	return true
}

// RequireRoles must run after Auth.
func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role := CurrentUserRole(c)
		if _, ok := allowed[role]; !ok {
			response.Abort(c, apperrors.Forbidden("you do not have permission to access this resource"))
			return
		}
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) string { return contextString(c, ContextUserID) }

func CurrentUserRole(c *gin.Context) string { return contextString(c, ContextUserRole) }

func contextString(c *gin.Context, key string) string {
	if value, ok := c.Get(key); ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
