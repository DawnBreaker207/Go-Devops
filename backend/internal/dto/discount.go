package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// DiscountCodeResponse is one code in the operator catalogue. It is NOT returned
// to a customer: applying a code answers with DiscountAppliedResponse instead, so
// the rules behind a code (minimum, remaining uses, window) never leak.
type DiscountCodeResponse struct {
	ID          string     `json:"id"`
	Code        string     `json:"code"`
	Description string     `json:"description,omitempty"`
	Kind        string     `json:"kind"`
	Value       int64      `json:"value"`
	MaxDiscount *int64     `json:"max_discount,omitempty"`
	MinOrder    int64      `json:"min_order"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	MaxUses     *int       `json:"max_uses,omitempty"`
	UsedCount   int        `json:"used_count"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewDiscountCodeResponse(d *models.DiscountCode) DiscountCodeResponse {
	return DiscountCodeResponse{
		ID:          d.ID,
		Code:        d.Code,
		Description: d.Description,
		Kind:        d.Kind,
		Value:       d.Value,
		MaxDiscount: d.MaxDiscount,
		MinOrder:    d.MinOrder,
		StartsAt:    d.StartsAt,
		EndsAt:      d.EndsAt,
		MaxUses:     d.MaxUses,
		UsedCount:   d.UsedCount,
		Active:      d.Active,
		CreatedAt:   d.CreatedAt,
	}
}

func NewDiscountCodeResponses(codes []models.DiscountCode) []DiscountCodeResponse {
	out := make([]DiscountCodeResponse, 0, len(codes))
	for i := range codes {
		out = append(out, NewDiscountCodeResponse(&codes[i]))
	}
	return out
}

// ApplyDiscountRequest is the customer-facing body of POST /orders/:id/discount.
type ApplyDiscountRequest struct {
	Code string `json:"code" binding:"required,min=1,max=32" example:"WELCOME10"`
}

// DiscountAppliedResponse is what the customer gets back: the three numbers the
// checkout screen needs and the code itself, nothing about the code's rules.
//
// `subtotal` is bookings.total_amount (the seat prices), `payable` is what the
// gateway will actually charge. A client that renders total_amount as the amount
// due after a discount is showing the wrong number.
type DiscountAppliedResponse struct {
	BookingID string `json:"booking_id"`
	Code      string `json:"code,omitempty"`
	Subtotal  int64  `json:"subtotal"`
	Discount  int64  `json:"discount"`
	Payable   int64  `json:"payable"`
}

func NewDiscountAppliedResponse(b *models.Booking, code string) DiscountAppliedResponse {
	return DiscountAppliedResponse{
		BookingID: b.ID,
		Code:      code,
		Subtotal:  b.TotalAmount,
		Discount:  b.DiscountAmount,
		Payable:   b.Payable(),
	}
}

/* --- Operator catalogue --- */

type DiscountListQuery struct {
	PageQuery
	// Go pointer: absent means no filter, distinct from false.
	Active *bool `form:"active"`
}

// CreateDiscountRequest adds a code. `kind` decides how `value` is read:
// percent (1..100, optionally capped by max_discount) or a flat amount in VND.
type CreateDiscountRequest struct {
	Code        string     `json:"code" binding:"required,min=3,max=32" example:"WELCOME10"`
	Description string     `json:"description" binding:"omitempty,max=2000"`
	Kind        string     `json:"kind" binding:"required,oneof=percent amount" example:"percent"`
	Value       int64      `json:"value" binding:"required,min=1" example:"10"`
	MaxDiscount *int64     `json:"max_discount" binding:"omitempty,min=1"`
	MinOrder    int64      `json:"min_order" binding:"min=0"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	MaxUses     *int       `json:"max_uses" binding:"omitempty,min=1"`
	// Omitted defaults to TRUE.
	Active *bool `json:"active"`
}

// UpdateDiscountRequest is a PARTIAL update; an omitted field is left alone.
// `code` and `kind` are absent on purpose: changing either would silently rewrite
// what a code meant for orders that already used it.
type UpdateDiscountRequest struct {
	Description *string    `json:"description" binding:"omitempty,max=2000"`
	Value       *int64     `json:"value" binding:"omitempty,min=1"`
	MaxDiscount *int64     `json:"max_discount" binding:"omitempty,min=1"`
	MinOrder    *int64     `json:"min_order" binding:"omitempty,min=0"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	MaxUses     *int       `json:"max_uses" binding:"omitempty,min=1"`
	Active      *bool      `json:"active"`
}

// IsEmpty reports a body that would change nothing, which the service rejects
// rather than issuing a no-op UPDATE.
func (r UpdateDiscountRequest) IsEmpty() bool {
	return r.Description == nil && r.Value == nil && r.MaxDiscount == nil &&
		r.MinOrder == nil && r.StartsAt == nil && r.EndsAt == nil &&
		r.MaxUses == nil && r.Active == nil
}
