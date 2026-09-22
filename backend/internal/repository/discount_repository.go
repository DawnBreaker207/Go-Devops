package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// DiscountRepository stores the discount-code catalogue and the one atomic
// counter that keeps a limited code from being over-redeemed.
type DiscountRepository interface {
	// FindByCode matches UPPERCASE and skips soft-deleted rows. Returns
	// (nil, nil) when there is no such code, leaving the sentinel to the service.
	FindByCode(ctx context.Context, code string) (*models.DiscountCode, error)
	FindByID(ctx context.Context, id string) (*models.DiscountCode, error)

	// ClaimUse increments used_count for one redemption and returns the number of
	// rows it touched. It is the ONLY guard against exceeding max_uses: the WHERE
	// carries the limit, so two concurrent applies cannot both win. 0 rows means
	// the code ran out between validation and this call.
	ClaimUse(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	// ReleaseUse gives a redemption back when an applied code is removed again.
	// Floored at 0 so a double-release can never drive the counter negative.
	ReleaseUse(ctx context.Context, tx *gorm.DB, id string) error

	List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.DiscountCode, int64, error)
	Create(ctx context.Context, tx *gorm.DB, code *models.DiscountCode) error
	Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error
	SoftDelete(ctx context.Context, tx *gorm.DB, id string) error
}

type discountRepository struct {
	db *gorm.DB
}

func NewDiscountRepository(db *gorm.DB) DiscountRepository {
	return &discountRepository{db: db}
}

func (r *discountRepository) FindByCode(ctx context.Context, code string) (*models.DiscountCode, error) {
	var found models.DiscountCode
	err := r.db.WithContext(ctx).
		Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find discount code: %w", err)
	}
	return &found, nil
}

func (r *discountRepository) FindByID(ctx context.Context, id string) (*models.DiscountCode, error) {
	var found models.DiscountCode
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find discount code: %w", err)
	}
	return &found, nil
}

func (r *discountRepository) ClaimUse(ctx context.Context, tx *gorm.DB, id string) (int64, error) {
	// The limit lives in the WHERE, not in a prior SELECT: that is what makes
	// two simultaneous redemptions of the last use impossible.
	res := tx.WithContext(ctx).Model(&models.DiscountCode{}).
		Where("id = ? AND (max_uses IS NULL OR used_count < max_uses)", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1"))
	if res.Error != nil {
		return 0, fmt.Errorf("claim discount use: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *discountRepository) ReleaseUse(ctx context.Context, tx *gorm.DB, id string) error {
	if err := tx.WithContext(ctx).Model(&models.DiscountCode{}).
		Where("id = ? AND used_count > 0", id).
		UpdateColumn("used_count", gorm.Expr("used_count - 1")).Error; err != nil {
		return fmt.Errorf("release discount use: %w", err)
	}
	return nil
}

func (r *discountRepository) List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.DiscountCode, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		if search != "" {
			pattern := "%" + strings.ToLower(search) + "%"
			tx = tx.Where("LOWER(code) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
		}
		if active != nil {
			tx = tx.Where("active = ?", *active)
		}
		return tx
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.DiscountCode{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count discount codes: %w", err)
	}

	var codes []models.DiscountCode
	if err := filter(r.db.WithContext(ctx)).
		Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&codes).Error; err != nil {
		return nil, 0, fmt.Errorf("list discount codes: %w", err)
	}
	return codes, total, nil
}

func (r *discountRepository) Create(ctx context.Context, tx *gorm.DB, code *models.DiscountCode) error {
	if err := tx.WithContext(ctx).Create(code).Error; err != nil {
		return fmt.Errorf("create discount code: %w", err)
	}
	return nil
}

func (r *discountRepository) Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	// A map, not a struct: only a map write can set a column to its zero value
	// (active false, min_order 0) on purpose.
	if err := tx.WithContext(ctx).Model(&models.DiscountCode{}).
		Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("update discount code: %w", err)
	}
	return nil
}

func (r *discountRepository) SoftDelete(ctx context.Context, tx *gorm.DB, id string) error {
	if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&models.DiscountCode{}).Error; err != nil {
		return fmt.Errorf("delete discount code: %w", err)
	}
	return nil
}
