package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	WaitlistWaiting   = "waiting"
	WaitlistNotified  = "notified"
	WaitlistFulfilled = "fulfilled"
	WaitlistExpired   = "expired"
	WaitlistCanceled  = "canceled"
)

type WaitlistEntry struct {
	ID                  string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID              string     `gorm:"type:uuid;not null" json:"user_id"`
	ShowtimeID          string     `gorm:"type:uuid;not null" json:"showtime_id"`
	Status              string     `gorm:"type:varchar(16);not null;default:waiting" json:"status"`
	FulfilledBookingID  *string    `gorm:"type:uuid" json:"fulfilled_booking_id,omitempty"`
	NotifiedAt          *time.Time `json:"notified_at,omitempty"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

func (WaitlistEntry) TableName() string { return "waitlist_entries" }

func (w *WaitlistEntry) BeforeCreate(*gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	return nil
}
