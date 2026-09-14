package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Showtime status.
const (
	ShowtimeOpen   = "open"
	ShowtimeClosed = "closed"
)

// Showtime schedules a movie in a hall at a time slot.
type Showtime struct {
	ID        string         `gorm:"type:uuid;primaryKey" json:"id"`
	MovieID   string         `gorm:"type:uuid;not null" json:"movie_id"`
	HallID    string         `gorm:"type:uuid;not null" json:"hall_id"`
	StartAt   time.Time      `gorm:"not null" json:"start_at"`
	EndAt     time.Time      `gorm:"not null" json:"end_at"`
	Status    string         `gorm:"type:varchar(16);not null;default:open" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Showtime) TableName() string { return "showtimes" }

func (s *Showtime) BeforeCreate(*gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

// Per-show seat status.
const (
	SeatStatusAvailable = "available"
	SeatStatusHeld      = "held"
	SeatStatusSold      = "sold"
)

// ShowtimeSeat is the state of one seat in one show — the locking and
// fencing point for holds. Version is the optimistic-lock token.
type ShowtimeSeat struct {
	ID         string     `gorm:"type:uuid;primaryKey" json:"id"`
	ShowtimeID string     `gorm:"type:uuid;not null" json:"showtime_id"`
	SeatID     string     `gorm:"type:uuid;not null" json:"seat_id"`
	Status     string     `gorm:"type:varchar(16);not null;default:available" json:"status"`
	HeldBy     *string    `gorm:"type:uuid" json:"held_by,omitempty"`
	HeldUntil  *time.Time `json:"held_until,omitempty"`
	Version    int64      `gorm:"not null;default:0" json:"version"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (ShowtimeSeat) TableName() string { return "showtime_seats" }

func (s *ShowtimeSeat) BeforeCreate(*gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}