package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type PricingRuleRepository struct {
	db *gorm.DB
}

func NewPricingRuleRepository(db *gorm.DB) *PricingRuleRepository {
	return &PricingRuleRepository{db: db}
}

func (r *PricingRuleRepository) Create(tx *gorm.DB, rule *models.PricingRule) error {
	return tx.Create(rule).Error
}

func (r *PricingRuleRepository) List(ctx context.Context, activeOnly bool) ([]models.PricingRule, error) {
	q := r.db.WithContext(ctx).Order("name")
	if activeOnly {
		q = q.Where("active = true")
	}
	var rows []models.PricingRule
	err := q.Find(&rows).Error
	return rows, err
}

func (r *PricingRuleRepository) FindByID(ctx context.Context, id string) (*models.PricingRule, error) {
	var rule models.PricingRule
	if err := r.db.WithContext(ctx).First(&rule, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *PricingRuleRepository) Update(tx *gorm.DB, rule *models.PricingRule) error {
	return tx.Model(rule).Select(
		"name", "day_of_week", "starts_at", "ends_at", "adjustment_type", "adjustment_value", "active", "updated_at",
	).Updates(rule).Error
}

// ActiveForHold is used inside the hold transaction: it takes tx so the read
// is consistent with the rest of the pricing computed in that transaction.
func (r *PricingRuleRepository) ActiveForHold(tx *gorm.DB) ([]models.PricingRule, error) {
	var rows []models.PricingRule
	err := tx.Where("active = true").Find(&rows).Error
	return rows, err
}
