package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	CreateTx(ctx context.Context, tx *gorm.DB, user *models.User) error
	FindByID(ctx context.Context, id string) (*models.User, error)
	StatusByID(ctx context.Context, id string) (active bool, role string, found bool, err error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context, query dto.UserListQuery) ([]models.User, int64, error)
	LockByID(ctx context.Context, tx *gorm.DB, id string) (*models.User, error)
	LockActiveAdmins(ctx context.Context, tx *gorm.DB) ([]string, error)
	SetActive(ctx context.Context, tx *gorm.DB, id string, active bool) error
	SetRole(ctx context.Context, tx *gorm.DB, id, role string) error
	SetPassword(ctx context.Context, tx *gorm.DB, id, hash string) error
	UpdateProfile(ctx context.Context, tx *gorm.DB, id, fullName, phone string) error
	SetAcceptedTerms(ctx context.Context, tx *gorm.DB, id string, version int) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	return r.CreateTx(ctx, r.db, user)
}

func (r *userRepository) CreateTx(ctx context.Context, tx *gorm.DB, user *models.User) error {
	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

// Reads only what the auth middleware checks on every request.
func (r *userRepository) StatusByID(ctx context.Context, id string) (bool, string, bool, error) {
	var rows []struct {
		Active bool
		Role   string
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT active, role FROM users WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&rows).Error; err != nil {
		return false, "", false, fmt.Errorf("read user status: %w", err)
	}
	if len(rows) == 0 {
		return false, "", false, nil
	}
	return rows[0].Active, rows[0].Role, true, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, fmt.Errorf("count user by email: %w", err)
	}
	return count > 0, nil
}

func (r *userRepository) List(ctx context.Context, query dto.UserListQuery) ([]models.User, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.User{})
	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(email) LIKE ? OR LOWER(full_name) LIKE ?", pattern, pattern)
	}
	if query.Role != "" {
		tx = tx.Where("role = ?", query.Role)
	}
	if query.Active != nil {
		tx = tx.Where("active = ?", *query.Active)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}
	users := make([]models.User, 0, query.PageSize)
	if err := tx.Order("created_at DESC").Limit(query.PageSize).Offset(query.Offset()).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (r *userRepository) LockByID(ctx context.Context, tx *gorm.DB, id string) (*models.User, error) {
	return firstOrNil[models.User](tx.WithContext(ctx).Clauses(forUpdate()).Where("id = ?", id), "lock user")
}

// Locks active admins in id order so concurrent lockouts serialize and can not remove the last admin.
func (r *userRepository) LockActiveAdmins(ctx context.Context, tx *gorm.DB) ([]string, error) {
	var ids []string
	if err := tx.WithContext(ctx).Raw(`SELECT id FROM users
		WHERE role = ? AND active AND deleted_at IS NULL ORDER BY id FOR UPDATE`, models.RoleAdmin).
		Scan(&ids).Error; err != nil {
		return nil, fmt.Errorf("lock active admins: %w", err)
	}
	return ids, nil
}

func (r *userRepository) SetRole(ctx context.Context, tx *gorm.DB, id, role string) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE users SET role = ?, updated_at = NOW() WHERE id = ?`, role, id).Error; err != nil {
		return fmt.Errorf("set user role: %w", err)
	}
	return nil
}

func (r *userRepository) SetActive(ctx context.Context, tx *gorm.DB, id string, active bool) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE users SET active = ?, updated_at = NOW() WHERE id = ?`, active, id).Error; err != nil {
		return fmt.Errorf("set user active: %w", err)
	}
	return nil
}

func (r *userRepository) SetPassword(ctx context.Context, tx *gorm.DB, id, hash string) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE users SET password = ?, updated_at = NOW() WHERE id = ?`, hash, id).Error; err != nil {
		return fmt.Errorf("set user password: %w", err)
	}
	return nil
}

// UpdateProfile: an empty phone clears the stored number.
func (r *userRepository) UpdateProfile(ctx context.Context, tx *gorm.DB, id, fullName, phone string) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE users SET full_name = ?, phone = NULLIF(?, ''), updated_at = NOW() WHERE id = ?`,
		fullName, phone, id).Error; err != nil {
		return fmt.Errorf("update user profile: %w", err)
	}
	return nil
}

func (r *userRepository) SetAcceptedTerms(ctx context.Context, tx *gorm.DB, id string, version int) error {
	if err := tx.WithContext(ctx).Exec(`UPDATE users SET accepted_terms_version = ?, updated_at = NOW() WHERE id = ?`, version, id).Error; err != nil {
		return fmt.Errorf("set accepted terms: %w", err)
	}
	return nil
}
