package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type BranchRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=255"`
	Address string `json:"address" binding:"omitempty,max=500"`
}

type UpdateBranchRequest struct {
	Name    *string `json:"name" binding:"omitempty,min=1,max=255"`
	Address *string `json:"address" binding:"omitempty,max=500"`
	Active  *bool   `json:"active"`
}

type BranchResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewBranchResponse(b *models.Branch) BranchResponse {
	return BranchResponse{ID: b.ID, Name: b.Name, Address: b.Address, Active: b.Active, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt}
}

func NewBranchResponses(rows []models.Branch) []BranchResponse {
	out := make([]BranchResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewBranchResponse(&rows[i]))
	}
	return out
}
