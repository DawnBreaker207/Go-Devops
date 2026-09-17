package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type BranchRepository struct {
	db *gorm.DB
}

func NewBranchRepository(db *gorm.DB) *BranchRepository {
	return &BranchRepository{db: db}
}

func (r *BranchRepository) Create(tx *gorm.DB, b *models.Branch) error {
	return tx.Create(b).Error
}

func (r *BranchRepository) List(ctx context.Context, activeOnly bool) ([]models.Branch, error) {
	q := r.db.WithContext(ctx).Order("name")
	if activeOnly {
		q = q.Where("active = true")
	}
	var rows []models.Branch
	err := q.Find(&rows).Error
	return rows, err
}

// Default is used to backfill Hall.BranchID when a create request omits it
// (keeps every pre-multi-branch caller, including this repo's own tests,
// working without having to name a branch explicitly).
func (r *BranchRepository) Default(ctx context.Context) (*models.Branch, error) {
	var b models.Branch
	if err := r.db.WithContext(ctx).Where("active = true").Order("created_at").First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *BranchRepository) FindByID(ctx context.Context, id string) (*models.Branch, error) {
	var b models.Branch
	if err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *BranchRepository) Update(tx *gorm.DB, b *models.Branch) error {
	return tx.Model(b).Select("name", "address", "active", "updated_at").Updates(b).Error
}

// HallBranch resolves a hall's branch_id, used to enforce a branch-scoped
// voucher (Phần 1.1) or a check-in gate at the wrong branch (Phần 4).
func (r *BranchRepository) HallBranch(tx *gorm.DB, hallID string) (string, error) {
	var branchID string
	err := tx.Raw(`SELECT branch_id FROM halls WHERE id = ?`, hallID).Scan(&branchID).Error
	return branchID, err
}
