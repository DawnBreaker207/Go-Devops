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

// DiscountService owns discount codes: the operator catalogue, and applying one
// to a customer's PENDING order.
//
// It never touches bookings.total_amount. That column is the seat subtotal and
// finalizeTx asserts the sold seats add up to it; a discounted total would make
// every discounted order fail to confirm AFTER the money was taken. The discount
// lives in its own column and reaches the gateway through Booking.Payable().
type DiscountService interface {
	Apply(ctx context.Context, userID, bookingID string, req dto.ApplyDiscountRequest) (*dto.DiscountAppliedResponse, error)
	Remove(ctx context.Context, userID, bookingID string) (*dto.DiscountAppliedResponse, error)

	AdminList(ctx context.Context, q dto.DiscountListQuery) ([]dto.DiscountCodeResponse, int64, error)
	AdminGet(ctx context.Context, id string) (*dto.DiscountCodeResponse, error)
	AdminCreate(ctx context.Context, req dto.CreateDiscountRequest) (*dto.DiscountCodeResponse, error)
	AdminUpdate(ctx context.Context, id string, req dto.UpdateDiscountRequest) (*dto.DiscountCodeResponse, error)
	AdminDelete(ctx context.Context, id string) error
}

type discountService struct {
	db       *gorm.DB
	codes    repository.DiscountRepository
	bookings repository.BookingRepository
	// now is injectable so the validity-window tests do not depend on wall clock.
	now func() time.Time
}

func NewDiscountService(db *gorm.DB, codes repository.DiscountRepository, bookings repository.BookingRepository) DiscountService {
	return &discountService{db: db, codes: codes, bookings: bookings, now: time.Now}
}

/* -------------------------------------------------------------------------- */
/* Customer: apply / remove                                                    */
/* -------------------------------------------------------------------------- */

// Apply attaches a code to a pending order and records what it took off.
//
// Every "no" answers a DISTINCT sentence but they all share code 40001, so the
// client shows the message rather than branching on a code. One deliberate
// exception: an UNKNOWN code answers the same ErrDiscountInvalid as a disabled
// one, so this endpoint cannot be used to enumerate which codes exist.
func (s *discountService) Apply(ctx context.Context, userID, bookingID string, req dto.ApplyDiscountRequest) (*dto.DiscountAppliedResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, apperrors.ErrDiscountInvalid
	}

	var (
		applied  models.Booking
		usedCode string
	)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the order first: the discount, the pay path and the sweep all
		// contend for this row.
		b, err := s.bookings.LockBooking(ctx, tx, bookingID)
		if err != nil {
			return err
		}
		if b == nil {
			return apperrors.ErrBookingNotFound
		}
		if b.UserID != userID {
			// 403 here, matching POST /orders/:id/refresh. (GET /tickets/:id/qr
			// answers 404 for the same situation; the two differ on purpose —
			// a ticket id must not be probeable, an order id the caller already
			// named is not a secret.)
			return apperrors.Forbidden("this order belongs to another user")
		}
		if b.Status != models.BookingPending || b.PaidAt != nil {
			return apperrors.ErrDiscountOrderClosed
		}
		if b.TotalAmount <= 0 {
			return apperrors.ErrBookingEmpty
		}
		if b.DiscountCodeID != nil {
			// Refuse rather than silently swapping: the customer must remove the
			// old code first, so the screen and the order never disagree.
			return apperrors.ErrDiscountAlreadySet
		}

		found, err := s.codes.FindByCode(ctx, code)
		if err != nil {
			return err
		}
		if err := s.validate(found, b.TotalAmount); err != nil {
			return err
		}

		off := found.DiscountFor(b.TotalAmount)
		if off <= 0 {
			return apperrors.ErrDiscountInvalid
		}
		// A 100%-off order would hand the gateway an amount of 0, which no
		// provider here is specified for. Refuse it as a bad code rather than
		// inventing a free-order path through the money code.
		if off >= b.TotalAmount {
			return apperrors.ErrDiscountInvalid
		}

		// Claim the use BEFORE writing the booking: the limit lives in the
		// UPDATE's WHERE, so two simultaneous redemptions of the last use cannot
		// both succeed.
		n, err := s.codes.ClaimUse(ctx, tx, found.ID)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.ErrDiscountExhausted
		}

		n, err = s.bookings.SetDiscount(ctx, tx, b.ID, &found.ID, off)
		if err != nil {
			return err
		}
		if n == 0 {
			// The order stopped being pending between the lock and here.
			return apperrors.ErrDiscountOrderClosed
		}

		b.DiscountAmount, b.DiscountCodeID = off, &found.ID
		applied, usedCode = *b, found.Code

		return s.auditDiscount(ctx, tx, "orders.apply_discount", b.ID, map[string]any{
			"code": found.Code, "discount": off, "payable": b.Payable(),
		})
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewDiscountAppliedResponse(&applied, usedCode)
	return &result, nil
}

// Remove clears the code and gives its redemption back.
func (s *discountService) Remove(ctx context.Context, userID, bookingID string) (*dto.DiscountAppliedResponse, error) {
	var cleared models.Booking

	err := s.db.Transaction(func(tx *gorm.DB) error {
		b, err := s.bookings.LockBooking(ctx, tx, bookingID)
		if err != nil {
			return err
		}
		if b == nil {
			return apperrors.ErrBookingNotFound
		}
		if b.UserID != userID {
			return apperrors.Forbidden("this order belongs to another user")
		}
		if b.Status != models.BookingPending || b.PaidAt != nil {
			return apperrors.ErrDiscountOrderClosed
		}
		if b.DiscountCodeID == nil {
			return apperrors.ErrDiscountNone
		}
		codeID := *b.DiscountCodeID

		n, err := s.bookings.SetDiscount(ctx, tx, b.ID, nil, 0)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.ErrDiscountOrderClosed
		}
		// Give the redemption back only after the booking write succeeded, so a
		// failure cannot leak a use.
		if err := s.codes.ReleaseUse(ctx, tx, codeID); err != nil {
			return err
		}

		b.DiscountAmount, b.DiscountCodeID = 0, nil
		cleared = *b

		return s.auditDiscount(ctx, tx, "orders.remove_discount", b.ID, map[string]any{
			"payable": b.Payable(),
		})
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewDiscountAppliedResponse(&cleared, "")
	return &result, nil
}

// validate answers why a code will not apply. An unknown code and a disabled one
// give the SAME answer on purpose (see Apply).
func (s *discountService) validate(d *models.DiscountCode, subtotal int64) error {
	if d == nil || !d.Active {
		return apperrors.ErrDiscountInvalid
	}
	now := s.now()
	if d.StartsAt != nil && now.Before(*d.StartsAt) {
		return apperrors.ErrDiscountNotStarted
	}
	if d.EndsAt != nil && !now.Before(*d.EndsAt) {
		return apperrors.ErrDiscountExpired
	}
	if d.MaxUses != nil && d.UsedCount >= *d.MaxUses {
		return apperrors.ErrDiscountExhausted
	}
	if subtotal < d.MinOrder {
		return apperrors.ErrDiscountMinOrder
	}
	return nil
}

func (s *discountService) auditDiscount(ctx context.Context, tx *gorm.DB, action, bookingID string, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return nil
	}
	rec.Action = action
	rec.ResourceType = "booking"
	rec.ResourceID = bookingID
	rec.BookingID = bookingID
	rec.After = after
	return audit.In(ctx, tx, rec)
}

/* -------------------------------------------------------------------------- */
/* Operator catalogue                                                          */
/* -------------------------------------------------------------------------- */

func (s *discountService) AdminList(ctx context.Context, q dto.DiscountListQuery) ([]dto.DiscountCodeResponse, int64, error) {
	codes, total, err := s.codes.List(ctx, q.Page, q.PageSize, strings.TrimSpace(q.Search), q.Active)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewDiscountCodeResponses(codes), total, nil
}

func (s *discountService) AdminGet(ctx context.Context, id string) (*dto.DiscountCodeResponse, error) {
	found, err := s.codes.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, apperrors.ErrDiscountNotFound
	}
	result := dto.NewDiscountCodeResponse(found)
	return &result, nil
}

func (s *discountService) AdminCreate(ctx context.Context, req dto.CreateDiscountRequest) (*dto.DiscountCodeResponse, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if err := validateDiscountShape(req.Kind, req.Value, req.MaxDiscount, req.StartsAt, req.EndsAt); err != nil {
		return nil, err
	}

	// Checked here for a friendly message; the partial unique index is what
	// actually guarantees it under a race.
	existing, err := s.codes.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperrors.ErrDiscountCodeExists
	}

	created := &models.DiscountCode{
		Code:        code,
		Description: strings.TrimSpace(req.Description),
		Kind:        req.Kind,
		Value:       req.Value,
		MaxDiscount: req.MaxDiscount,
		MinOrder:    req.MinOrder,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		MaxUses:     req.MaxUses,
		Active:      req.Active == nil || *req.Active,
	}
	if created.Kind == models.DiscountAmount {
		// max_discount only caps a percentage; keeping it on a flat code would
		// read as a second, silent limit.
		created.MaxDiscount = nil
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.codes.Create(ctx, tx, created); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrDiscountCodeExists
			}
			return err
		}
		return s.auditCode(ctx, tx, created.ID, nil, discountAuditFields(created))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewDiscountCodeResponse(created)
	return &result, nil
}

func (s *discountService) AdminUpdate(ctx context.Context, id string, req dto.UpdateDiscountRequest) (*dto.DiscountCodeResponse, error) {
	if req.IsEmpty() {
		return nil, apperrors.Validation("nothing to update")
	}

	current, err := s.codes.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, apperrors.ErrDiscountNotFound
	}
	before := discountAuditFields(current)

	fields := make(map[string]any, 8)
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
		fields["description"] = current.Description
	}
	if req.Value != nil {
		current.Value = *req.Value
		fields["value"] = current.Value
	}
	if req.MaxDiscount != nil {
		current.MaxDiscount = req.MaxDiscount
		fields["max_discount"] = current.MaxDiscount
	}
	if req.MinOrder != nil {
		current.MinOrder = *req.MinOrder
		fields["min_order"] = current.MinOrder
	}
	if req.StartsAt != nil {
		current.StartsAt = req.StartsAt
		fields["starts_at"] = current.StartsAt
	}
	if req.EndsAt != nil {
		current.EndsAt = req.EndsAt
		fields["ends_at"] = current.EndsAt
	}
	if req.MaxUses != nil {
		current.MaxUses = req.MaxUses
		fields["max_uses"] = current.MaxUses
	}
	if req.Active != nil {
		current.Active = *req.Active
		fields["active"] = current.Active
	}

	// Re-validate the RESULT, not the request: a percentage that was legal before
	// can be made illegal by changing only `value`.
	if err := validateDiscountShape(current.Kind, current.Value, current.MaxDiscount, current.StartsAt, current.EndsAt); err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.codes.Update(ctx, tx, id, fields); err != nil {
			return err
		}
		return s.auditCode(ctx, tx, id, before, discountAuditFields(current))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewDiscountCodeResponse(current)
	return &result, nil
}

// AdminDelete soft-deletes: bookings.discount_code_id still points here, and the
// partial unique index frees the string for reuse afterwards.
func (s *discountService) AdminDelete(ctx context.Context, id string) error {
	current, err := s.codes.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return apperrors.ErrDiscountNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.codes.SoftDelete(ctx, tx, id); err != nil {
			return err
		}
		return s.auditCode(ctx, tx, id, discountAuditFields(current), nil)
	})
}

// validateDiscountShape holds the rules the `binding` tags cannot express,
// because they are relationships between fields rather than field ranges.
func validateDiscountShape(kind string, value int64, maxDiscount *int64, startsAt, endsAt *time.Time) error {
	if kind == models.DiscountPercent && (value < 1 || value > 100) {
		return apperrors.Validation("a percentage discount must be between 1 and 100")
	}
	if kind == models.DiscountAmount && maxDiscount != nil {
		return apperrors.Validation("max_discount only applies to a percentage code")
	}
	if startsAt != nil && endsAt != nil && !endsAt.After(*startsAt) {
		return apperrors.Validation("ends_at must be after starts_at")
	}
	return nil
}

func discountAuditFields(d *models.DiscountCode) map[string]any {
	return map[string]any{
		"code": d.Code, "kind": d.Kind, "value": d.Value, "active": d.Active,
	}
}

func (s *discountService) auditCode(ctx context.Context, tx *gorm.DB, id string, before, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return nil
	}
	rec.ResourceID = id
	rec.Before = before
	rec.After = after
	return audit.In(ctx, tx, rec)
}
