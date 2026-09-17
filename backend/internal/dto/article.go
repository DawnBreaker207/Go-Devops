package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type ArticleRequest struct {
	Title        string `json:"title" binding:"required,min=1,max=255" example:"Khuyen mai thang 10"`
	// Auto-generated from title when omitted (a numeric suffix is appended on a collision).
	Slug string `json:"slug" binding:"omitempty,max=255" example:"khuyen-mai-thang-10"`
	Summary      string `json:"summary" binding:"omitempty,max=500"`
	ThumbnailURL string `json:"thumbnail_url" binding:"omitempty,max=1024"`
	Content      string `json:"content" binding:"required"`
	Type         string `json:"type" binding:"omitempty,oneof=news promotion" example:"news"`
	Status       string `json:"status" binding:"omitempty,oneof=draft published hidden" example:"draft"`
}

// UpdateArticleRequest: omitted fields keep their current value.
type UpdateArticleRequest struct {
	Title        *string `json:"title" binding:"omitempty,min=1,max=255"`
	Slug         *string `json:"slug" binding:"omitempty,max=255"`
	Summary      *string `json:"summary" binding:"omitempty,max=500"`
	ThumbnailURL *string `json:"thumbnail_url" binding:"omitempty,max=1024"`
	Content      *string `json:"content" binding:"omitempty,min=1"`
	Type         *string `json:"type" binding:"omitempty,oneof=news promotion"`
	Status       *string `json:"status" binding:"omitempty,oneof=draft published hidden"`
}

type ArticleResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	Summary      string    `json:"summary,omitempty"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	Content      string    `json:"content"`
	AuthorID     string    `json:"author_id"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Views        int64     `json:"views"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewArticleResponse(a *models.Article) ArticleResponse {
	return ArticleResponse{
		ID: a.ID, Title: a.Title, Slug: a.Slug, Summary: a.Summary, ThumbnailURL: a.ThumbnailURL,
		Content: a.Content, AuthorID: a.AuthorID, Type: a.Type, Status: a.Status, Views: a.Views,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func NewArticleResponses(rows []models.Article) []ArticleResponse {
	out := make([]ArticleResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewArticleResponse(&rows[i]))
	}
	return out
}
