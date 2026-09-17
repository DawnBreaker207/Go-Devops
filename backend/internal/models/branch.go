package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Branch struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Address   string    `gorm:"type:varchar(500)" json:"address,omitempty"`
	Active    bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Branch) TableName() string { return "branches" }

func (b *Branch) BeforeCreate(*gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}
