package service

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type PricingRuleService interface {
	List(ctx context.Context, activeOnly bool) ([]dto.PricingRuleResponse, error)
	Create(ctx context.Context, req dto.PricingRuleRequest) (*dto.PricingRuleResponse, error)
	Update(ctx context.Context, id string, req dto.UpdatePricingRuleRequest) (*dto.PricingRuleResponse, error)
}

type pricingRuleService struct {
	db   *gorm.DB
	repo *repository.PricingRuleRepository
}

func NewPricingRuleService(db *gorm.DB, repo *repository.PricingRuleRepository) PricingRuleService {
	return &pricingRuleService{db: db, repo: repo}
}

func (s *pricingRuleService) List(ctx context.Context, activeOnly bool) ([]dto.PricingRuleResponse, error) {
	rows, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return dto.NewPricingRuleResponses(rows), nil
}

func validateWindow(startsAt, endsAt string) error {
	if (startsAt == "") != (endsAt == "") {
		return apperrors.Validation("starts_at and ends_at must both be set or both omitted")
	}
	return nil
}

func (s *pricingRuleService) Create(ctx context.Context, req dto.PricingRuleRequest) (*dto.PricingRuleResponse, error) {
	if err := validateWindow(req.StartsAt, req.EndsAt); err != nil {
		return nil, err
	}
	rule := &models.PricingRule{
		Name: strings.TrimSpace(req.Name), DayOfWeek: req.DayOfWeek,
		AdjustmentType: req.AdjustmentType, AdjustmentValue: req.AdjustmentValue, Active: true,
	}
	if req.StartsAt != "" {
		rule.StartsAt, rule.EndsAt = &req.StartsAt, &req.EndsAt
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, rule); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = rule.ID
			rec.After = map[string]any{"name": rule.Name, "adjustment_type": rule.AdjustmentType, "adjustment_value": rule.AdjustmentValue}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewPricingRuleResponse(rule)
	return &result, nil
}

func (s *pricingRuleService) Update(ctx context.Context, id string, req dto.UpdatePricingRuleRequest) (*dto.PricingRuleResponse, error) {
	var rule *models.PricingRule
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.NotFound("pricing rule not found")
		}
		before := map[string]any{"adjustment_value": current.AdjustmentValue, "active": current.Active}

		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.DayOfWeek != nil {
			current.DayOfWeek = req.DayOfWeek
		}
		if req.StartsAt != nil {
			current.StartsAt = req.StartsAt
		}
		if req.EndsAt != nil {
			current.EndsAt = req.EndsAt
		}
		startsAt, endsAt := "", ""
		if current.StartsAt != nil {
			startsAt = *current.StartsAt
		}
		if current.EndsAt != nil {
			endsAt = *current.EndsAt
		}
		if err := validateWindow(startsAt, endsAt); err != nil {
			return err
		}
		if req.AdjustmentType != nil {
			current.AdjustmentType = *req.AdjustmentType
		}
		if req.AdjustmentValue != nil {
			current.AdjustmentValue = *req.AdjustmentValue
		}
		if req.Active != nil {
			current.Active = *req.Active
		}
		if err := s.repo.Update(tx, current); err != nil {
			return err
		}
		rule = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"adjustment_value": rule.AdjustmentValue, "active": rule.Active}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewPricingRuleResponse(rule)
	return &result, nil
}
