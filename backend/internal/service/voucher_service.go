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

// VoucherService: creation and approval are separate actions (a voucher is
// created pending_approval and only takes effect once moved to active) —
// groundwork for the maker-checker step of Phần 9.3, not yet a hard 2-person
// requirement at this scale (Phần 9.6 "để sau, chỉ khi có nhiều admin thật").
type VoucherService interface {
	List(ctx context.Context, status string, query dto.PageQuery) ([]dto.VoucherResponse, int64, error)
	GetByID(ctx context.Context, id string) (*dto.VoucherResponse, error)
	Create(ctx context.Context, createdByAdminID string, req dto.VoucherRequest) (*dto.VoucherResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateVoucherRequest) (*dto.VoucherResponse, error)
	SetStatus(ctx context.Context, actorID, id, status string) (*dto.VoucherResponse, error)
	// Conflicts lists redemptions where the redeeming user is also the
	// voucher's own creator (Phần 9.4 conflict-of-interest detection).
	Conflicts(ctx context.Context) ([]dto.VoucherConflictResponse, error)
}

type voucherService struct {
	db   *gorm.DB
	repo *repository.VoucherRepository
}

func NewVoucherService(db *gorm.DB, repo *repository.VoucherRepository) VoucherService {
	return &voucherService{db: db, repo: repo}
}

func (s *voucherService) List(ctx context.Context, status string, query dto.PageQuery) ([]dto.VoucherResponse, int64, error) {
	rows, total, err := s.repo.List(ctx, status, query.PageSize, query.Offset())
	if err != nil {
		return nil, 0, err
	}
	return dto.NewVoucherResponses(rows), total, nil
}

func (s *voucherService) GetByID(ctx context.Context, id string) (*dto.VoucherResponse, error) {
	v, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, apperrors.ErrVoucherNotFound
	}
	result := dto.NewVoucherResponse(v)
	return &result, nil
}

func (s *voucherService) Create(ctx context.Context, createdByAdminID string, req dto.VoucherRequest) (*dto.VoucherResponse, error) {
	if !req.EndsAt.After(req.StartsAt) {
		return nil, apperrors.Validation("ends_at must be after starts_at")
	}
	scope := req.ApplyScope
	if scope == "" {
		scope = models.VoucherScopeAll
	}
	perUser := req.MaxUsagePerUser
	if perUser == 0 {
		perUser = 1
	}
	v := &models.Voucher{
		Code: strings.ToUpper(strings.TrimSpace(req.Code)), DiscountType: req.DiscountType,
		DiscountValue: req.DiscountValue, MaxDiscount: req.MaxDiscount, MinOrderAmount: req.MinOrderAmount,
		StartsAt: req.StartsAt, EndsAt: req.EndsAt, MaxUsage: req.MaxUsage, MaxUsagePerUser: perUser,
		ApplyScope: scope, CreatedByAdminID: &createdByAdminID, Status: models.VoucherPendingApproval,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, v); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrVoucherCodeExists
			}
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = v.ID
			rec.After = map[string]any{"code": v.Code, "discount_type": v.DiscountType, "discount_value": v.DiscountValue}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewVoucherResponse(v)
	return &result, nil
}

// Update is only allowed while the voucher has never gone live
// (status=pending_approval): once active, its terms must not shift under
// customers already relying on them.
func (s *voucherService) Update(ctx context.Context, id string, req dto.UpdateVoucherRequest) (*dto.VoucherResponse, error) {
	var v *models.Voucher
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrVoucherNotFound
		}
		if current.Status != models.VoucherPendingApproval {
			return apperrors.ErrVoucherNotEditable
		}
		before := map[string]any{"discount_value": current.DiscountValue, "max_usage": current.MaxUsage}

		if req.DiscountType != nil {
			current.DiscountType = *req.DiscountType
		}
		if req.DiscountValue != nil {
			current.DiscountValue = *req.DiscountValue
		}
		if req.MaxDiscount != nil {
			current.MaxDiscount = req.MaxDiscount
		}
		if req.MinOrderAmount != nil {
			current.MinOrderAmount = *req.MinOrderAmount
		}
		if req.StartsAt != nil {
			current.StartsAt = *req.StartsAt
		}
		if req.EndsAt != nil {
			current.EndsAt = *req.EndsAt
		}
		if req.MaxUsage != nil {
			current.MaxUsage = *req.MaxUsage
		}
		if req.MaxUsagePerUser != nil {
			current.MaxUsagePerUser = *req.MaxUsagePerUser
		}
		if req.ApplyScope != nil {
			current.ApplyScope = *req.ApplyScope
		}
		if !current.EndsAt.After(current.StartsAt) {
			return apperrors.Validation("ends_at must be after starts_at")
		}
		if err := s.repo.Update(tx, current); err != nil {
			return err
		}
		v = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"discount_value": v.DiscountValue, "max_usage": v.MaxUsage}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewVoucherResponse(v)
	return &result, nil
}

// SetStatus moves a voucher between pending_approval -> active/rejected, or
// active/pending_approval -> disabled (a manual kill switch at any time).
// SetStatus moving a voucher to active is maker-checker (Phần 9.3): the
// approver must be a different admin than the one who created it, so a
// single admin can never both create and activate a discount alone.
func (s *voucherService) SetStatus(ctx context.Context, actorID, id, status string) (*dto.VoucherResponse, error) {
	var v *models.Voucher
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrVoucherNotFound
		}
		if status == models.VoucherActive && current.CreatedByAdminID != nil && *current.CreatedByAdminID == actorID {
			return apperrors.ErrVoucherSelfApproval
		}
		before := current.Status
		n, err := s.repo.SetStatus(tx, id, status)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.ErrVoucherNotFound
		}
		current.Status = status
		v = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = map[string]any{"status": before}
			rec.After = map[string]any{"status": status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewVoucherResponse(v)
	return &result, nil
}

func (s *voucherService) Conflicts(ctx context.Context) ([]dto.VoucherConflictResponse, error) {
	rows, err := s.repo.Conflicts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.VoucherConflictResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.VoucherConflictResponse{
			VoucherID: r.VoucherID, Code: r.Code, AdminID: r.AdminID, BookingID: r.BookingID, UsedAt: r.UsedAt,
		})
	}
	return out, nil
}
