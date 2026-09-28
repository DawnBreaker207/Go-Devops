package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Groups code, combo and article under one date window; no soft delete.
type Campaign struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	StartsAt    time.Time `gorm:"not null" json:"starts_at"`
	EndsAt      time.Time `gorm:"not null" json:"ends_at"`
	// No `default:` tag: GORM omits zero-valued fields with a default, silently re-enabling active=false.
	Active bool `gorm:"not null" json:"active"`
	// v1 only enforces exactly 1 via discount_redemptions' UNIQUE(user_id, discount_code_id).
	PerUserLimit int       `gorm:"not null;default:1" json:"per_user_limit"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Campaign) TableName() string { return "campaigns" }

func (c *Campaign) BeforeCreate(*gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

// Active && now in [StartsAt, EndsAt); flipped by hand, no cron.
func (c *Campaign) InWindow(now time.Time) bool {
	return c.Active && !now.Before(c.StartsAt) && now.Before(c.EndsAt)
}

// PromoPrice never overrides the real sell price.
type CampaignCombo struct {
	CampaignID string `gorm:"type:uuid;primaryKey" json:"campaign_id"`
	ComboID    string `gorm:"type:uuid;primaryKey" json:"combo_id"`
	PromoPrice *int64 `json:"promo_price,omitempty"`
}

func (CampaignCombo) TableName() string { return "campaign_combos" }

// Linking alone doesn't imply visibility; public reads also require ArticlePublished.
type CampaignArticle struct {
	CampaignID string `gorm:"type:uuid;primaryKey" json:"campaign_id"`
	ArticleID  string `gorm:"type:uuid;primaryKey" json:"article_id"`
}

func (CampaignArticle) TableName() string { return "campaign_articles" }

// DiscountRedemption is the per-account-once guard; its UNIQUE constraint is the concurrency control.
type DiscountRedemption struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         string    `gorm:"type:uuid;not null" json:"user_id"`
	DiscountCodeID string    `gorm:"type:uuid;not null" json:"discount_code_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (DiscountRedemption) TableName() string { return "discount_redemptions" }

func (d *DiscountRedemption) BeforeCreate(*gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}
