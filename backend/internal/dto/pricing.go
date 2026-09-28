package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Send all four keys or just the one being changed.
type BasePriceRequest struct {
	Prices map[string]int64 `json:"prices" binding:"required" example:"standard:80000,vip:120000"`
}

type BasePriceResponse struct {
	SeatType  string    `json:"seat_type"`
	Price     int64     `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewBasePriceResponse(p *models.SeatBasePrice) BasePriceResponse {
	return BasePriceResponse{SeatType: p.SeatType, Price: p.Price, UpdatedAt: p.UpdatedAt}
}

func NewBasePriceResponses(prices []models.SeatBasePrice) []BasePriceResponse {
	out := make([]BasePriceResponse, 0, len(prices))
	for i := range prices {
		out = append(out, NewBasePriceResponse(&prices[i]))
	}
	return out
}

// from_price is the cheapest CONFIGURED one.
type PublicPriceListResponse struct {
	FromPrice int64            `json:"from_price"`
	Prices    map[string]int64 `json:"prices"`
}

// Times are "HH:MM"/"HH:MM:SS", date is "YYYY-MM-DD".
// A row with day_of_week, start_time, end_time and specific_date all omitted matches everything.
type CreatePricingRuleRequest struct {
	Name         string  `json:"name" binding:"required,min=1,max=255" example:"Weekend surcharge"`
	DayOfWeek    *int    `json:"day_of_week" binding:"omitempty,min=0,max=6" example:"0"`
	StartTime    *string `json:"start_time" example:"18:00"`
	EndTime      *string `json:"end_time" example:"23:00"`
	SpecificDate *string `json:"specific_date" example:"2026-01-01"`
	AdjustKind   string  `json:"adjust_kind" binding:"required,oneof=percent fixed" example:"percent"`
	// Signed: negative discounts, positive surcharges.
	AdjustValue int64 `json:"adjust_value" example:"10"`
	Priority    int   `json:"priority" binding:"omitempty,min=0"`
	// Omitted defaults to TRUE.
	Active *bool `json:"active"`
}

// A set field can't be cleared back to nil.
type UpdatePricingRuleRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=1,max=255"`
	DayOfWeek    *int    `json:"day_of_week" binding:"omitempty,min=0,max=6"`
	StartTime    *string `json:"start_time"`
	EndTime      *string `json:"end_time"`
	SpecificDate *string `json:"specific_date"`
	AdjustKind   *string `json:"adjust_kind" binding:"omitempty,oneof=percent fixed"`
	AdjustValue  *int64  `json:"adjust_value"`
	Priority     *int    `json:"priority" binding:"omitempty,min=0"`
	Active       *bool   `json:"active"`
}

// The service rejects a no-op UPDATE.
func (r UpdatePricingRuleRequest) IsEmpty() bool {
	return r.Name == nil && r.DayOfWeek == nil && r.StartTime == nil && r.EndTime == nil &&
		r.SpecificDate == nil && r.AdjustKind == nil && r.AdjustValue == nil &&
		r.Priority == nil && r.Active == nil
}

type PricingRuleListQuery struct {
	PageQuery
	Active *bool `form:"active"`
}

type PricingRuleResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DayOfWeek    *int      `json:"day_of_week,omitempty"`
	StartTime    *string   `json:"start_time,omitempty"`
	EndTime      *string   `json:"end_time,omitempty"`
	SpecificDate *string   `json:"specific_date,omitempty"`
	AdjustKind   string    `json:"adjust_kind"`
	AdjustValue  int64     `json:"adjust_value"`
	Priority     int       `json:"priority"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewPricingRuleResponse(r *models.PricingRule) PricingRuleResponse {
	out := PricingRuleResponse{
		ID:          r.ID,
		Name:        r.Name,
		AdjustKind:  r.AdjustKind,
		AdjustValue: r.AdjustValue,
		Priority:    r.Priority,
		Active:      r.Active,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.DayOfWeek != nil {
		dow := int(*r.DayOfWeek)
		out.DayOfWeek = &dow
	}
	if r.StartTime != nil {
		s := string(*r.StartTime)
		out.StartTime = &s
	}
	if r.EndTime != nil {
		s := string(*r.EndTime)
		out.EndTime = &s
	}
	if r.SpecificDate != nil {
		s := r.SpecificDate.Format("2006-01-02")
		out.SpecificDate = &s
	}
	return out
}

func NewPricingRuleResponses(rules []models.PricingRule) []PricingRuleResponse {
	out := make([]PricingRuleResponse, 0, len(rules))
	for i := range rules {
		out = append(out, NewPricingRuleResponse(&rules[i]))
	}
	return out
}

type PricingQuoteQuery struct {
	ShowtimeID string `form:"showtime_id" binding:"required,uuid"`
	SeatType   string `form:"seat_type" binding:"required,oneof=standard vip couple recliner"`
}

// In the order applied.
type AppliedPricingRule struct {
	RuleID      string `json:"rule_id"`
	Name        string `json:"name"`
	AdjustKind  string `json:"adjust_kind"`
	AdjustValue int64  `json:"adjust_value"`
}

// Read-only breakdown, floored at 0.
type PricingQuoteResponse struct {
	Base    int64                `json:"base"`
	Applied []AppliedPricingRule `json:"applied"`
	Final   int64                `json:"final"`
}
