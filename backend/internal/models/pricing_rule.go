package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PricingAdjustPercent = "percent"
	PricingAdjustFixed   = "fixed"
)

// PricingRule: a day-of-week + time-of-day window surcharge/discount over
// hall_prices (rule-based, Product Backlog "giá động theo ngày/khung giờ").
type PricingRule struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string    `gorm:"type:varchar(255);not null" json:"name"`
	DayOfWeek       *int      `json:"day_of_week,omitempty"`
	StartsAt        *string   `gorm:"type:time" json:"starts_at,omitempty"`
	EndsAt          *string   `gorm:"type:time" json:"ends_at,omitempty"`
	AdjustmentType  string    `gorm:"type:varchar(16);not null" json:"adjustment_type"`
	AdjustmentValue float64   `gorm:"type:numeric(10,2);not null" json:"adjustment_value"`
	Active          bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (PricingRule) TableName() string { return "pricing_rules" }

func (p *PricingRule) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
