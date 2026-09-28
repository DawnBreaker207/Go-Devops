package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// New values need the Go const, the `oneof=` DTO tag and the SQL CHECK.
const (
	// Multiplies the running total by adjust_value/100 (see PricingService.Quote).
	AdjustPercent = "percent"
	AdjustFixed = "fixed"
)

var AllAdjustKinds = []string{AdjustPercent, AdjustFixed}

// Wall-clock "HH:MM:SS" over SQL TIME.
type ClockTime string

func (c *ClockTime) Scan(value any) error {
	if value == nil {
		*c = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*c = ClockTime(v)
	case []byte:
		*c = ClockTime(string(v))
	case time.Time:
		*c = ClockTime(v.Format("15:04:05"))
	default:
		return fmt.Errorf("cannot scan %T into ClockTime", value)
	}
	return nil
}

func (c ClockTime) Value() (driver.Value, error) {
	if c == "" {
		return nil, nil
	}
	return string(c), nil
}

// ONE global price per seat type; booking flow doesn't read it yet.
type SeatBasePrice struct {
	SeatType  string    `gorm:"column:seat_type;type:text;primaryKey" json:"seat_type"`
	Price     int64     `gorm:"not null" json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SeatBasePrice) TableName() string { return "seat_base_prices" }

// All-nil scope matches everything.
type PricingRule struct {
	ID   string `gorm:"type:uuid;primaryKey" json:"id"`
	Name string `gorm:"type:text;not null" json:"name"`
	// 0 = Sunday .. 6 = Saturday, matching time.Weekday. Nil = every day.
	DayOfWeek *int16 `gorm:"type:smallint" json:"day_of_week,omitempty"`
	// Nil/nil = matches all day.
	StartTime *ClockTime `gorm:"type:time" json:"start_time,omitempty"`
	EndTime   *ClockTime `gorm:"type:time" json:"end_time,omitempty"`
	SpecificDate *time.Time `gorm:"type:date" json:"specific_date,omitempty"`
	AdjustKind   string     `gorm:"type:text;not null" json:"adjust_kind"`
	// Signed: negative is a discount, positive a surcharge.
	AdjustValue int64     `gorm:"not null" json:"adjust_value"`
	Priority    int       `gorm:"not null;default:0" json:"priority"`
	Active      bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (PricingRule) TableName() string { return "pricing_rules" }

func (r *PricingRule) BeforeCreate(*gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}
