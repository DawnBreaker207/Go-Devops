package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type PricingRuleRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
	// 0=Sunday..6=Saturday; omitted = every day.
	DayOfWeek *int `json:"day_of_week" binding:"omitempty,min=0,max=6"`
	// "HH:MM" local time; both or neither.
	StartsAt        string  `json:"starts_at" binding:"omitempty,len=5"`
	EndsAt          string  `json:"ends_at" binding:"omitempty,len=5"`
	AdjustmentType  string  `json:"adjustment_type" binding:"required,oneof=percent fixed"`
	AdjustmentValue float64 `json:"adjustment_value" binding:"required"`
}

type UpdatePricingRuleRequest struct {
	Name            *string  `json:"name" binding:"omitempty,min=1,max=255"`
	DayOfWeek       *int     `json:"day_of_week" binding:"omitempty,min=0,max=6"`
	StartsAt        *string  `json:"starts_at" binding:"omitempty,len=5"`
	EndsAt          *string  `json:"ends_at" binding:"omitempty,len=5"`
	AdjustmentType  *string  `json:"adjustment_type" binding:"omitempty,oneof=percent fixed"`
	AdjustmentValue *float64 `json:"adjustment_value"`
	Active          *bool    `json:"active"`
}

type PricingRuleResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	DayOfWeek       *int      `json:"day_of_week,omitempty"`
	StartsAt        string    `json:"starts_at,omitempty"`
	EndsAt          string    `json:"ends_at,omitempty"`
	AdjustmentType  string    `json:"adjustment_type"`
	AdjustmentValue float64   `json:"adjustment_value"`
	Active          bool      `json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func NewPricingRuleResponse(r *models.PricingRule) PricingRuleResponse {
	res := PricingRuleResponse{
		ID: r.ID, Name: r.Name, DayOfWeek: r.DayOfWeek,
		AdjustmentType: r.AdjustmentType, AdjustmentValue: r.AdjustmentValue,
		Active: r.Active, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.StartsAt != nil {
		res.StartsAt = *r.StartsAt
	}
	if r.EndsAt != nil {
		res.EndsAt = *r.EndsAt
	}
	return res
}

func NewPricingRuleResponses(rows []models.PricingRule) []PricingRuleResponse {
	out := make([]PricingRuleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewPricingRuleResponse(&rows[i]))
	}
	return out
}
