package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Combo struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	Price       int64     `gorm:"not null" json:"price"`
	MemberPrice *int64    `json:"member_price,omitempty"`
	Active      bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Combo) TableName() string { return "combos" }

func (c *Combo) BeforeCreate(*gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// BookingCombo: per-unit price snapshot at hold time, same convention as BookingSeat.Price.
type BookingCombo struct {
	ID          string     `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID   string     `gorm:"type:uuid;not null" json:"booking_id"`
	ComboID     string     `gorm:"type:uuid;not null" json:"combo_id"`
	Quantity    int        `gorm:"not null" json:"quantity"`
	Price       int64      `gorm:"not null" json:"price"`
	Delivered   bool       `gorm:"not null;default:false" json:"delivered"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (BookingCombo) TableName() string { return "booking_combos" }

func (c *BookingCombo) BeforeCreate(*gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}
