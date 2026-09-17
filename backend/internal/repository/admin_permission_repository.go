package repository

import (
	"context"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type AdminPermissionRepository struct {
	db *gorm.DB
}

func NewAdminPermissionRepository(db *gorm.DB) *AdminPermissionRepository {
	return &AdminPermissionRepository{db: db}
}

// Has is the deny-by-default check the permission middleware runs on every
// gated admin request: no row means no access, even for role=admin.
func (r *AdminPermissionRepository) Has(ctx context.Context, userID, key string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AdminPermission{}).
		Where("user_id = ? AND permission_key = ?", userID, key).Count(&count).Error
	return count > 0, err
}

func (r *AdminPermissionRepository) ListByUser(ctx context.Context, userID string) ([]models.AdminPermission, error) {
	var rows []models.AdminPermission
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	return rows, err
}

func (r *AdminPermissionRepository) Grant(tx *gorm.DB, p *models.AdminPermission) error {
	return tx.Create(p).Error
}

func (r *AdminPermissionRepository) Revoke(tx *gorm.DB, userID, key string) (int64, error) {
	res := tx.Where("user_id = ? AND permission_key = ?", userID, key).Delete(&models.AdminPermission{})
	return res.RowsAffected, res.Error
}

func (r *AdminPermissionRepository) RevokeAll(tx *gorm.DB, userID string) error {
	return tx.Where("user_id = ?", userID).Delete(&models.AdminPermission{}).Error
}

// CountActiveOwners backs the "at least one owner must remain" invariant,
// mirroring UserRepository.CountActiveAdmins for the admin role.
func (r *AdminPermissionRepository) CountActiveOwners(ctx context.Context, tx *gorm.DB) (int64, error) {
	conn := r.db.WithContext(ctx)
	if tx != nil {
		conn = tx
	}
	var count int64
	err := conn.Model(&models.User{}).Where("role = ? AND active = true AND deleted_at IS NULL", models.RoleOwner).Count(&count).Error
	return count, err
}
