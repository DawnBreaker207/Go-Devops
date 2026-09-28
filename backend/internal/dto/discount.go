package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Operator-only; customers get DiscountAppliedResponse instead.
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
	CampaignID *string   `json:"campaign_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
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
		CampaignID:  d.CampaignID,
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

type ApplyDiscountRequest struct {
	Code string `json:"code" binding:"required,min=1,max=32" example:"WELCOME10"`
}

// Subtotal is bookings.total_amount; payable is what the gateway charges.
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

type DiscountListQuery struct {
	PageQuery
	Active *bool `form:"active"`
}

// kind decides how value is read: percent (1..100) or flat VND.
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

// code and kind are immutable.
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

// The service rejects a no-op UPDATE.
func (r UpdateDiscountRequest) IsEmpty() bool {
	return r.Description == nil && r.Value == nil && r.MaxDiscount == nil &&
		r.MinOrder == nil && r.StartsAt == nil && r.EndsAt == nil &&
		r.MaxUses == nil && r.Active == nil
}
