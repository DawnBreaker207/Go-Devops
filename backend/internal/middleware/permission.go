package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// PermissionChecker reads admin_permissions (Phần 9.2 of
// ADVANCED_FEATURES_DISCUSSION.md).
type PermissionChecker interface {
	Has(ctx context.Context, userID, permissionKey string) (bool, error)
}

// RequirePermission is deny-by-default: role=admin alone is not enough for a
// pricing/finance/accounts-gated route — a fresh admin has zero rows in
// admin_permissions and is refused until an owner grants the key. role=owner
// always passes, since owner is the one who grants these groups and must not
// be locked out by its own grant table being empty. Must run after Auth and
// after RequireRoles(RoleAdmin, RoleOwner).
func RequirePermission(checker PermissionChecker, permissionKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUserRole(c) == models.RoleOwner {
			c.Next()
			return
		}
		ok, err := checker.Has(c.Request.Context(), CurrentUserID(c), permissionKey)
		if err != nil {
			response.Abort(c, apperrors.From(err))
			return
		}
		if !ok {
			response.Abort(c, apperrors.ErrPermissionDenied)
			return
		}
		c.Next()
	}
}
