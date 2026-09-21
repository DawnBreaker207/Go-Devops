package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MovieStatusDraft      = "draft"
	MovieStatusComingSoon = "coming_soon"
	MovieStatusShowing    = "showing"
	MovieStatusEnded      = "ended"
)

type Movie struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title       string         `gorm:"type:varchar(255);not null;index" json:"title"`
	Genre       string         `gorm:"type:varchar(100);not null" json:"genre"`
	Duration    int            `gorm:"not null" json:"duration"`
	Director    string         `gorm:"type:varchar(255);not null" json:"director"`
	Description string         `gorm:"type:text" json:"description"`
	PosterURL   string         `gorm:"type:varchar(512)" json:"poster_url"`
	TrailerURL  string         `gorm:"type:varchar(512)" json:"trailer_url"`
	Cast        string         `gorm:"column:cast_members;type:text" json:"cast"`
	AgeRating   string         `gorm:"type:varchar(4);not null;default:P" json:"age_rating"`
	ReleaseDate time.Time      `gorm:"type:date;not null" json:"release_date"`
	Status      string         `gorm:"type:varchar(32);not null;default:draft;index" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Movie) TableName() string { return "movies" }

func (m *Movie) BeforeCreate(*gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
