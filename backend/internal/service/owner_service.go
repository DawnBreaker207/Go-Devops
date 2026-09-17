package service

import (
	"context"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// OwnerService: role=owner is a tier above admin (Phần 9.1 of
// ADVANCED_FEATURES_DISCUSSION.md) — it grants/revokes the four admin
// permission groups and promotes/demotes admin<->owner, but is not itself
// granted through the generic admin user-update endpoint, so an admin can
// never escalate itself or a peer to owner.
type OwnerService interface {
	SetOwner(ctx context.Context, actorID, userID string, owner bool) (*dto.UserResponse, error)
	Grant(ctx context.Context, actorID, userID, permissionKey string) error
	Revoke(ctx context.Context, actorID, userID, permissionKey string) error
	ListPermissions(ctx context.Context, userID string) ([]dto.AdminPermissionResponse, error)
}

type ownerService struct {
	db       *gorm.DB
	userRepo repository.UserRepository
	permRepo *repository.AdminPermissionRepository
	onChange []func(userID string)
}

func NewOwnerService(db *gorm.DB, userRepo repository.UserRepository, permRepo *repository.AdminPermissionRepository, onChange ...func(userID string)) OwnerService {
	return &ownerService{db: db, userRepo: userRepo, permRepo: permRepo, onChange: onChange}
}

// SetOwner promotes an active admin to owner, or demotes an owner back to a
// (permission-less) admin. The system must always keep at least one active
// owner (Phần 9.1 "không cho phép khoá/xoá owner cuối cùng").
func (s *ownerService) SetOwner(ctx context.Context, actorID, userID string, owner bool) (*dto.UserResponse, error) {
	var user *models.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		owners, err := s.userRepo.LockActiveOwners(ctx, tx)
		if err != nil {
			return err
		}
		target, err := s.userRepo.LockByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if target == nil {
			return apperrors.ErrUserNotFound
		}

		before := map[string]any{"role": target.Role}
		switch {
		case owner:
			if target.Role != models.RoleAdmin {
				return apperrors.Validation("only an active admin can be promoted to owner")
			}
			if err := s.userRepo.SetRole(ctx, tx, target.ID, models.RoleOwner); err != nil {
				return err
			}
			target.Role = models.RoleOwner
		default:
			if target.Role != models.RoleOwner {
				return apperrors.Validation("user is not an owner")
			}
			// Demoting always leaves the owner role, so an active owner can
			// only step down while at least one other active owner remains.
			if target.Active && len(owners) <= 1 {
				return apperrors.ErrLastOwner
			}
			if err := s.userRepo.SetRole(ctx, tx, target.ID, models.RoleAdmin); err != nil {
				return err
			}
			target.Role = models.RoleAdmin
			// A demoted owner keeps no dangling elevated grants: it re-enters
			// admin under deny-by-default, same as any freshly created admin.
			if err := s.permRepo.RevokeAll(tx, target.ID); err != nil {
				return err
			}
		}
		user = target
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = target.ID
			rec.Before = before
			rec.After = map[string]any{"role": target.Role}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, fn := range s.onChange {
		fn(userID)
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}

// Grant adds one permission group to an admin; deny-by-default means a fresh
// admin account has none of the four until an owner calls this.
func (s *ownerService) Grant(ctx context.Context, actorID, userID, permissionKey string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := s.userRepo.LockByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if target == nil {
			return apperrors.ErrUserNotFound
		}
		if target.Role != models.RoleAdmin {
			return apperrors.Validation("permissions can only be granted to an admin account")
		}
		perm := &models.AdminPermission{UserID: userID, PermissionKey: permissionKey, GrantedBy: actorID}
		if err := s.permRepo.Grant(tx, perm); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return nil // already granted: idempotent
			}
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = userID
			rec.After = map[string]any{"permission_key": permissionKey, "granted_by": actorID}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}

func (s *ownerService) Revoke(ctx context.Context, actorID, userID, permissionKey string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		n, err := s.permRepo.Revoke(tx, userID, permissionKey)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = userID
			rec.After = map[string]any{"permission_key": permissionKey, "revoked_by": actorID}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}

func (s *ownerService) ListPermissions(ctx context.Context, userID string) ([]dto.AdminPermissionResponse, error) {
	rows, err := s.permRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.AdminPermissionResponse, 0, len(rows))
	for _, p := range rows {
		result = append(result, dto.AdminPermissionResponse{
			UserID: p.UserID, PermissionKey: p.PermissionKey, GrantedBy: p.GrantedBy, CreatedAt: p.CreatedAt,
		})
	}
	return result, nil
}
