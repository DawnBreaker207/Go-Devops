package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SeatStandard = "standard"
	SeatVIP      = "vip"
	SeatCouple   = "couple"
	SeatRecliner = "recliner"
)

var AllSeatTypes = []string{SeatStandard, SeatVIP, SeatCouple, SeatRecliner}

const (
	ScreenFront = "front"
	ScreenBack  = "back"
)

type Hall struct {
	ID             string              `gorm:"type:uuid;primaryKey" json:"id"`
	Name           string              `gorm:"type:varchar(255);not null" json:"name"`
	Rows           int                 `gorm:"not null" json:"rows"`
	SeatsPerRow    int                 `gorm:"not null" json:"seats_per_row"`
	SeatTypes      map[string][]string `gorm:"serializer:json;type:jsonb;not null" json:"seat_types"`
	Gaps           []string            `gorm:"serializer:json;type:jsonb;not null" json:"gaps"`
	ScreenPosition string              `gorm:"type:varchar(8);not null;default:front" json:"screen_position"`
	AisleAfterCols []int               `gorm:"serializer:json;type:jsonb;not null" json:"aisle_after_cols"`
	Active         bool                `gorm:"not null;default:true" json:"active"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	DeletedAt      gorm.DeletedAt      `gorm:"index" json:"-"`
}

func (Hall) TableName() string { return "halls" }

func (h *Hall) BeforeCreate(*gorm.DB) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	return nil
}

type Seat struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	HallID    string    `gorm:"type:uuid;not null;uniqueIndex:uq_seat_hall_row_col" json:"hall_id"`
	RowIndex  int       `gorm:"not null" json:"row_index"`
	RowLabel  string    `gorm:"type:varchar(8);not null;uniqueIndex:uq_seat_hall_row_col" json:"row_label"`
	ColNumber int       `gorm:"not null;uniqueIndex:uq_seat_hall_row_col" json:"col_number"`
	SeatType  string    `gorm:"type:varchar(16);not null;default:standard" json:"seat_type"`
	IsGap     bool      `gorm:"not null;default:false" json:"is_gap"`
	ColSpan   int       `gorm:"type:smallint;not null;default:1" json:"col_span"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Seat) TableName() string { return "seats" }

func (s *Seat) BeforeCreate(*gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

type HallPrice struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	HallID    string    `gorm:"type:uuid;not null;uniqueIndex:uq_hall_prices_seat_type" json:"hall_id"`
	SeatType  string    `gorm:"type:varchar(16);not null;uniqueIndex:uq_hall_prices_seat_type" json:"seat_type"`
	Price     int64     `gorm:"not null" json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (HallPrice) TableName() string { return "hall_prices" }

func (p *HallPrice) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
