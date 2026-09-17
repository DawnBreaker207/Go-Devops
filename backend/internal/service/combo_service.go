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

type ComboService interface {
	List(ctx context.Context, activeOnly bool, query dto.PageQuery) ([]dto.ComboResponse, int64, error)
	GetByID(ctx context.Context, id string) (*dto.ComboResponse, error)
	Create(ctx context.Context, req dto.ComboRequest) (*dto.ComboResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateComboRequest) (*dto.ComboResponse, error)
	// SetStock caps a combo's stock at one branch; quantity is the absolute
	// new value, not a delta (Phần 2.2 "admin cập nhật tay").
	SetStock(ctx context.Context, branchID, comboID string, quantity int) error
}

type comboService struct {
	db   *gorm.DB
	repo *repository.ComboRepository
}

func NewComboService(db *gorm.DB, repo *repository.ComboRepository) ComboService {
	return &comboService{db: db, repo: repo}
}

func (s *comboService) List(ctx context.Context, activeOnly bool, query dto.PageQuery) ([]dto.ComboResponse, int64, error) {
	combos, total, err := s.repo.List(ctx, activeOnly, query.PageSize, query.Offset())
	if err != nil {
		return nil, 0, err
	}
	return dto.NewComboResponses(combos), total, nil
}

func (s *comboService) GetByID(ctx context.Context, id string) (*dto.ComboResponse, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperrors.ErrComboNotFound
	}
	result := dto.NewComboResponse(c)
	return &result, nil
}

func (s *comboService) Create(ctx context.Context, req dto.ComboRequest) (*dto.ComboResponse, error) {
	if req.MemberPrice != nil && *req.MemberPrice > req.Price {
		return nil, apperrors.Validation("member_price can not exceed price")
	}
	combo := &models.Combo{
		Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description),
		Price: req.Price, MemberPrice: req.MemberPrice, Active: true,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, combo); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = combo.ID
			rec.After = map[string]any{"name": combo.Name, "price": combo.Price, "member_price": combo.MemberPrice}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewComboResponse(combo)
	return &result, nil
}

func (s *comboService) Update(ctx context.Context, id string, req dto.UpdateComboRequest) (*dto.ComboResponse, error) {
	var combo *models.Combo
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrComboNotFound
		}
		before := map[string]any{"price": current.Price, "member_price": current.MemberPrice, "active": current.Active}

		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Description != nil {
			current.Description = strings.TrimSpace(*req.Description)
		}
		if req.Price != nil {
			current.Price = *req.Price
		}
		if req.MemberPrice != nil {
			current.MemberPrice = req.MemberPrice
		}
		if req.Active != nil {
			current.Active = *req.Active
		}
		if current.MemberPrice != nil && *current.MemberPrice > current.Price {
			return apperrors.Validation("member_price can not exceed price")
		}
		if err := s.repo.Update(tx, current); err != nil {
			return err
		}
		combo = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"price": combo.Price, "member_price": combo.MemberPrice, "active": combo.Active}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewComboResponse(combo)
	return &result, nil
}

func (s *comboService) SetStock(ctx context.Context, branchID, comboID string, quantity int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		combo, err := s.repo.FindByID(ctx, comboID)
		if err != nil {
			return err
		}
		if combo == nil {
			return apperrors.ErrComboNotFound
		}
		if err := s.repo.SetStock(tx, branchID, comboID, quantity); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = comboID
			rec.After = map[string]any{"branch_id": branchID, "stock_quantity": quantity}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}
