package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationPreference is one opt-in row per user; a missing row reads as defaults.
// First GET creates the row so a later PUT has something to update; both flags may be off.
type NotificationPreference struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID           string    `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	BookingReminders bool      `gorm:"not null;default:true" json:"booking_reminders"`
	PromoOffers      bool      `gorm:"not null;default:true" json:"promo_offers"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (NotificationPreference) TableName() string { return "notification_preferences" }

func (n *NotificationPreference) BeforeCreate(*gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}
