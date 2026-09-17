package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type MembershipService interface {
	ListTiers(ctx context.Context, activeOnly bool) ([]dto.MembershipTierResponse, error)
	CreateTier(ctx context.Context, req dto.MembershipTierRequest) (*dto.MembershipTierResponse, error)
	UpdateTier(ctx context.Context, id string, req dto.UpdateMembershipTierRequest) (*dto.MembershipTierResponse, error)
	// Purchase renews (extends expires_at) when the caller already holds an
	// active membership of the same tier; a different tier while still active
	// is refused (Phần 1.2 — no silent tier switch, no retroactive discount).
	Purchase(ctx context.Context, userID string, req dto.PurchaseMembershipRequest) (*dto.UserMembershipResponse, error)
	MyMemberships(ctx context.Context, userID string) ([]dto.UserMembershipResponse, error)
}

type membershipServiceImpl struct {
	db     *gorm.DB
	repo   *repository.MembershipRepository
	ledger *repository.LedgerRepository
}

func NewMembershipService(db *gorm.DB, repo *repository.MembershipRepository, ledger *repository.LedgerRepository) MembershipService {
	return &membershipServiceImpl{db: db, repo: repo, ledger: ledger}
}

func (s *membershipServiceImpl) ListTiers(ctx context.Context, activeOnly bool) ([]dto.MembershipTierResponse, error) {
	tiers, err := s.repo.ListTiers(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return dto.NewMembershipTierResponses(tiers), nil
}

func (s *membershipServiceImpl) CreateTier(ctx context.Context, req dto.MembershipTierRequest) (*dto.MembershipTierResponse, error) {
	tier := &models.MembershipTier{
		Name: strings.TrimSpace(req.Name), Price: req.Price, DurationDays: req.DurationDays,
		DiscountPercent: req.DiscountPercent, ExcludedSeatTypes: req.ExcludedSeatTypes, Active: true,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateTier(tx, tier); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = tier.ID
			rec.After = map[string]any{"name": tier.Name, "price": tier.Price, "discount_percent": tier.DiscountPercent}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewMembershipTierResponse(tier)
	return &result, nil
}

func (s *membershipServiceImpl) UpdateTier(ctx context.Context, id string, req dto.UpdateMembershipTierRequest) (*dto.MembershipTierResponse, error) {
	var tier *models.MembershipTier
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindTierByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrMembershipTierNotFound
		}
		before := map[string]any{"price": current.Price, "discount_percent": current.DiscountPercent, "active": current.Active}

		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.Price != nil {
			current.Price = *req.Price
		}
		if req.DurationDays != nil {
			current.DurationDays = *req.DurationDays
		}
		if req.DiscountPercent != nil {
			current.DiscountPercent = *req.DiscountPercent
		}
		if req.ExcludedSeatTypes != nil {
			current.ExcludedSeatTypes = req.ExcludedSeatTypes
		}
		if req.Active != nil {
			current.Active = *req.Active
		}
		if err := s.repo.UpdateTier(tx, current); err != nil {
			return err
		}
		tier = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"price": tier.Price, "discount_percent": tier.DiscountPercent, "active": tier.Active}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewMembershipTierResponse(tier)
	return &result, nil
}

// Purchase: idempotency_key is required (Phần 1.2 "Cần idempotency_key cho
// API mua gói"); a retry with the same key returns the same membership
// instead of double-charging or double-extending.
func (s *membershipServiceImpl) Purchase(ctx context.Context, userID string, req dto.PurchaseMembershipRequest) (*dto.UserMembershipResponse, error) {
	if !req.AcceptTerms {
		return nil, apperrors.ErrTermsRequired
	}
	key := strings.TrimSpace(req.IdempotencyKey)

	var (
		membership *models.UserMembership
		tier       *models.MembershipTier
	)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if existing, err := s.repo.FindByIdempotencyKey(tx, key); err != nil {
			return err
		} else if existing != nil {
			membership = existing
			t, err := s.repo.FindTierByID(ctx, existing.TierID)
			if err != nil {
				return err
			}
			if t == nil {
				return apperrors.ErrMembershipTierNotFound
			}
			tier = t
			return nil
		}

		t, err := s.repo.FindTierByID(ctx, req.TierID)
		if err != nil {
			return err
		}
		if t == nil {
			return apperrors.ErrMembershipTierNotFound
		}
		if !t.Active {
			return apperrors.ErrMembershipTierInactive
		}
		tier = t

		now := time.Now()
		active, err := s.repo.LockActiveByUser(tx, userID)
		if err != nil {
			return err
		}
		if active != nil {
			if active.TierID != req.TierID {
				return apperrors.ErrMembershipAlreadyActive
			}
			// Renew: extend from the current expiry (still to come) or from now
			// (already lapsed) — Phần 1.2.
			base := now
			if active.ExpiresAt.After(now) {
				base = active.ExpiresAt
			}
			newExpiry := base.AddDate(0, 0, tier.DurationDays)
			if err := s.repo.ExtendExpiry(tx, active.ID, newExpiry); err != nil {
				return err
			}
			active.ExpiresAt = newExpiry
			membership = active
		} else {
			m := &models.UserMembership{
				UserID: userID, TierID: tier.ID, StartedAt: now,
				ExpiresAt: now.AddDate(0, 0, tier.DurationDays), Status: models.MembershipActive,
			}
			if key != "" {
				m.IdempotencyKey = &key
			}
			if err := s.repo.Create(tx, m); err != nil {
				return err
			}
			membership = m
		}
		if s.ledger != nil {
			if err := s.ledger.Write(tx, &models.LedgerEntry{
				Type: models.LedgerMembershipPurchase, Amount: tier.Price,
				ReferenceTable: "user_memberships", ReferenceID: membership.ID, UserID: &userID,
			}); err != nil {
				return err
			}
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = membership.ID
			rec.After = map[string]any{"tier_id": tier.ID, "expires_at": membership.ExpiresAt}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewUserMembershipResponse(membership, tier)
	return &result, nil
}

func (s *membershipServiceImpl) MyMemberships(ctx context.Context, userID string) ([]dto.UserMembershipResponse, error) {
	rows, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.UserMembershipResponse, 0, len(rows))
	for i := range rows {
		tier, err := s.repo.FindTierByID(ctx, rows[i].TierID)
		if err != nil {
			return nil, err
		}
		if tier == nil {
			continue
		}
		out = append(out, dto.NewUserMembershipResponse(&rows[i], tier))
	}
	return out, nil
}
