package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// PricingRepository stores global base price per seat type plus adjustment rules.
type PricingRepository interface {
	GetBasePrices(ctx context.Context) ([]models.SeatBasePrice, error)
	// UpsertBasePrice sets one seat type's global price, creating the row the
	// first time it is set for a seat type the backfill left at its default.
	UpsertBasePrice(ctx context.Context, tx *gorm.DB, seatType string, price int64) error

	ListRules(ctx context.Context, query dto.PricingRuleListQuery) ([]models.PricingRule, int64, error)
	// GetRule returns (nil, nil) when there is no such rule, leaving the
	// sentinel to the service.
	GetRule(ctx context.Context, id string) (*models.PricingRule, error)
// ListActiveRules: every active rule; Quote matches day/time itself (single place).
	ListActiveRules(ctx context.Context) ([]models.PricingRule, error)
	CreateRule(ctx context.Context, tx *gorm.DB, rule *models.PricingRule) error
	UpdateRule(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error
	DeleteRule(ctx context.Context, tx *gorm.DB, id string) error
}

type pricingRepository struct {
	db *gorm.DB
}

func NewPricingRepository(db *gorm.DB) PricingRepository {
	return &pricingRepository{db: db}
}

func (r *pricingRepository) GetBasePrices(ctx context.Context) ([]models.SeatBasePrice, error) {
	var prices []models.SeatBasePrice
	if err := r.db.WithContext(ctx).Order("seat_type").Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("list base prices: %w", err)
	}
	return prices, nil
}

func (r *pricingRepository) UpsertBasePrice(ctx context.Context, tx *gorm.DB, seatType string, price int64) error {
	row := models.SeatBasePrice{SeatType: seatType, Price: price}
	err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "seat_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"price", "updated_at"}),
	}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert base price: %w", err)
	}
	return nil
}

func (r *pricingRepository) ListRules(ctx context.Context, query dto.PricingRuleListQuery) ([]models.PricingRule, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		if query.Search != "" {
			tx = tx.Where("LOWER(name) LIKE ?", "%"+query.Search+"%")
		}
		if query.Active != nil {
			tx = tx.Where("active = ?", *query.Active)
		}
		return tx
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.PricingRule{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pricing rules: %w", err)
	}

	var rules []models.PricingRule
	if err := filter(r.db.WithContext(ctx)).
		Order("priority DESC, created_at DESC").
		Limit(query.PageSize).Offset(query.Offset()).
		Find(&rules).Error; err != nil {
		return nil, 0, fmt.Errorf("list pricing rules: %w", err)
	}
	return rules, total, nil
}

func (r *pricingRepository) GetRule(ctx context.Context, id string) (*models.PricingRule, error) {
	var found models.PricingRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find pricing rule: %w", err)
	}
	return &found, nil
}

func (r *pricingRepository) ListActiveRules(ctx context.Context) ([]models.PricingRule, error) {
	var rules []models.PricingRule
	if err := r.db.WithContext(ctx).Where("active = ?", true).
		Order("priority DESC, created_at ASC").Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("list active pricing rules: %w", err)
	}
	return rules, nil
}

func (r *pricingRepository) CreateRule(ctx context.Context, tx *gorm.DB, rule *models.PricingRule) error {
	if err := tx.WithContext(ctx).Create(rule).Error; err != nil {
		return fmt.Errorf("create pricing rule: %w", err)
	}
	return nil
}

func (r *pricingRepository) UpdateRule(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Model(&models.PricingRule{}).
		Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("update pricing rule: %w", err)
	}
	return nil
}

func (r *pricingRepository) DeleteRule(ctx context.Context, tx *gorm.DB, id string) error {
// Hard delete: no other row references a rule (Quote returns snapshot, no FK).
	if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&models.PricingRule{}).Error; err != nil {
		return fmt.Errorf("delete pricing rule: %w", err)
	}
	return nil
}
