package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type MembershipTierRequest struct {
	Name              string   `json:"name" binding:"required,min=1,max=128" example:"Gold"`
	Price             int64    `json:"price" binding:"required,min=1" example:"499000"`
	DurationDays      int      `json:"duration_days" binding:"required,min=1" example:"365"`
	DiscountPercent   float64  `json:"discount_percent" binding:"required,gt=0,lte=100" example:"10"`
	ExcludedSeatTypes []string `json:"excluded_seat_types" binding:"omitempty,dive,oneof=standard vip couple recliner"`
}

type UpdateMembershipTierRequest struct {
	Name              *string  `json:"name" binding:"omitempty,min=1,max=128"`
	Price             *int64   `json:"price" binding:"omitempty,min=1"`
	DurationDays      *int     `json:"duration_days" binding:"omitempty,min=1"`
	DiscountPercent   *float64 `json:"discount_percent" binding:"omitempty,gt=0,lte=100"`
	ExcludedSeatTypes []string `json:"excluded_seat_types" binding:"omitempty,dive,oneof=standard vip couple recliner"`
	Active            *bool    `json:"active"`
}

type MembershipTierResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Price             int64     `json:"price"`
	DurationDays      int       `json:"duration_days"`
	DiscountPercent   float64   `json:"discount_percent"`
	ExcludedSeatTypes []string  `json:"excluded_seat_types,omitempty"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewMembershipTierResponse(t *models.MembershipTier) MembershipTierResponse {
	return MembershipTierResponse{
		ID: t.ID, Name: t.Name, Price: t.Price, DurationDays: t.DurationDays,
		DiscountPercent: t.DiscountPercent, ExcludedSeatTypes: t.ExcludedSeatTypes,
		Active: t.Active, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func NewMembershipTierResponses(rows []models.MembershipTier) []MembershipTierResponse {
	out := make([]MembershipTierResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewMembershipTierResponse(&rows[i]))
	}
	return out
}

// PurchaseMembershipRequest: buying while already having an active
// membership renews it (extends expires_at) instead of erroring, as long as
// the tier matches; a different tier while still active is refused (Phần 1.2
// "mua giữa chừng không có tác dụng hồi tố" — renew must be an explicit,
// deliberate action, not a silent tier switch).
type PurchaseMembershipRequest struct {
	TierID         string `json:"tier_id" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required,min=1,max=128"`
	AcceptTerms    bool   `json:"accept_terms" binding:"required"`
}

type UserMembershipResponse struct {
	ID        string                  `json:"id"`
	TierID    string                  `json:"tier_id"`
	Tier      MembershipTierResponse  `json:"tier"`
	StartedAt time.Time               `json:"started_at"`
	ExpiresAt time.Time               `json:"expires_at"`
	Status    string                  `json:"status"`
}

func NewUserMembershipResponse(m *models.UserMembership, tier *models.MembershipTier) UserMembershipResponse {
	return UserMembershipResponse{
		ID: m.ID, TierID: m.TierID, Tier: NewMembershipTierResponse(tier),
		StartedAt: m.StartedAt, ExpiresAt: m.ExpiresAt, Status: m.Status,
	}
}
