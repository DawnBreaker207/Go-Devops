package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Missing row reads as defaults; first GET creates it.
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
