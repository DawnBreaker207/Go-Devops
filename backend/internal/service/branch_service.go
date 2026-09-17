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

type BranchService interface {
	List(ctx context.Context, activeOnly bool) ([]dto.BranchResponse, error)
	GetByID(ctx context.Context, id string) (*dto.BranchResponse, error)
	Create(ctx context.Context, req dto.BranchRequest) (*dto.BranchResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateBranchRequest) (*dto.BranchResponse, error)
}

type branchService struct {
	db   *gorm.DB
	repo *repository.BranchRepository
}

func NewBranchService(db *gorm.DB, repo *repository.BranchRepository) BranchService {
	return &branchService{db: db, repo: repo}
}

func (s *branchService) List(ctx context.Context, activeOnly bool) ([]dto.BranchResponse, error) {
	rows, err := s.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return dto.NewBranchResponses(rows), nil
}

func (s *branchService) GetByID(ctx context.Context, id string) (*dto.BranchResponse, error) {
	b, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, apperrors.NotFound("branch not found")
	}
	result := dto.NewBranchResponse(b)
	return &result, nil
}

func (s *branchService) Create(ctx context.Context, req dto.BranchRequest) (*dto.BranchResponse, error) {
	b := &models.Branch{Name: strings.TrimSpace(req.Name), Address: strings.TrimSpace(req.Address), Active: true}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, b); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.Conflict("branch name already exists")
			}
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = b.ID
			rec.After = map[string]any{"name": b.Name}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewBranchResponse(b)
	return &result, nil
}

func (s *branchService) Update(ctx context.Context, id string, req dto.UpdateBranchRequest) (*dto.BranchResponse, error) {
	var b *models.Branch
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.NotFound("branch not found")
		}
		before := map[string]any{"name": current.Name, "active": current.Active}
		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Address != nil {
			current.Address = strings.TrimSpace(*req.Address)
		}
		if req.Active != nil {
			current.Active = *req.Active
		}
		if err := s.repo.Update(tx, current); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.Conflict("branch name already exists")
			}
			return err
		}
		b = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"name": b.Name, "active": b.Active}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewBranchResponse(b)
	return &result, nil
}
