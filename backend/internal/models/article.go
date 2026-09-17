package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ArticleTypeNews      = "news"
	ArticleTypePromotion = "promotion"
)

const (
	ArticleDraft     = "draft"
	ArticlePublished = "published"
	ArticleHidden    = "hidden"
)

type Article struct {
	ID           string         `gorm:"type:uuid;primaryKey" json:"id"`
	Title        string         `gorm:"type:varchar(255);not null" json:"title"`
	Slug         string         `gorm:"type:varchar(255);not null" json:"slug"`
	Summary      string         `gorm:"type:varchar(500)" json:"summary,omitempty"`
	ThumbnailURL string         `gorm:"type:varchar(1024)" json:"thumbnail_url,omitempty"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	AuthorID     string         `gorm:"type:uuid;not null" json:"author_id"`
	Type         string         `gorm:"type:varchar(16);not null;default:news" json:"type"`
	Status       string         `gorm:"type:varchar(16);not null;default:draft" json:"status"`
	Views        int64          `gorm:"not null;default:0" json:"views"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Article) TableName() string { return "articles" }

func (a *Article) BeforeCreate(*gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
