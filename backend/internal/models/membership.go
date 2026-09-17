package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MembershipActive  = "active"
	MembershipExpired = "expired"
)

type MembershipTier struct {
	ID                string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name              string    `gorm:"type:varchar(128);not null" json:"name"`
	Price             int64     `gorm:"not null" json:"price"`
	DurationDays      int       `gorm:"not null" json:"duration_days"`
	DiscountPercent   float64   `gorm:"type:numeric(5,2);not null" json:"discount_percent"`
	ExcludedSeatTypes []string  `gorm:"type:jsonb;serializer:json" json:"excluded_seat_types,omitempty"`
	Active            bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (MembershipTier) TableName() string { return "membership_tiers" }

func (t *MembershipTier) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

// At most one ACTIVE membership per user (partial unique index).
type UserMembership struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         string    `gorm:"type:uuid;not null" json:"user_id"`
	TierID         string    `gorm:"type:uuid;not null" json:"tier_id"`
	StartedAt      time.Time `json:"started_at"`
	ExpiresAt      time.Time `gorm:"not null" json:"expires_at"`
	Status         string     `gorm:"type:varchar(16);not null;default:active" json:"status"`
	IdempotencyKey *string    `gorm:"type:varchar(128)" json:"idempotency_key,omitempty"`
	ReminderSentAt *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (UserMembership) TableName() string { return "user_memberships" }

func (m *UserMembership) BeforeCreate(*gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
