package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// Keys storing authenticated user info in the gin context.
const (
	ContextUserID    = "user_id"
	ContextUserEmail = "user_email"
	ContextUserRole  = "user_role"
)

// Auth validates the access token in the Authorization: Bearer <token> header.
func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Abort(c, apperrors.Unauthorized("authorization header is required"))
			return
		}

		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Abort(c, apperrors.Unauthorized("authorization header must be in format: Bearer <token>"))
			return
		}

		claims, err := jwtManager.ParseAccess(parts[1])
		if err != nil {
			response.Abort(c, err)
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserEmail, claims.Email)
		c.Set(ContextUserRole, claims.Role)
		c.Next()
	}
}

// RequireRoles allows only the listed roles. Must run after Auth.
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

// CurrentUserID returns the authenticated user's id.
func CurrentUserID(c *gin.Context) string { return contextString(c, ContextUserID) }

// CurrentUserRole returns the authenticated user's role.
func CurrentUserRole(c *gin.Context) string { return contextString(c, ContextUserRole) }

func contextString(c *gin.Context, key string) string {
	if value, ok := c.Get(key); ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}
