package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type ComboRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255" example:"Combo bap nuoc lon"`
	Description string `json:"description" binding:"omitempty,max=2000"`
	Price       int64  `json:"price" binding:"required,min=1" example:"89000"`
	// Fixed member price, not a percentage of Price (Phần 1.3).
	MemberPrice *int64 `json:"member_price" binding:"omitempty,min=1" example:"79000"`
}

type UpdateComboRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Price       *int64  `json:"price" binding:"omitempty,min=1"`
	MemberPrice *int64  `json:"member_price" binding:"omitempty,min=1"`
	Active      *bool   `json:"active"`
}

type ComboResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Price       int64     `json:"price"`
	MemberPrice *int64    `json:"member_price,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewComboResponse(c *models.Combo) ComboResponse {
	return ComboResponse{
		ID: c.ID, Name: c.Name, Description: c.Description, Price: c.Price,
		MemberPrice: c.MemberPrice, Active: c.Active, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func NewComboResponses(rows []models.Combo) []ComboResponse {
	out := make([]ComboResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewComboResponse(&rows[i]))
	}
	return out
}

// ComboItem is a line of the combo cart sent at hold time (dto.HoldRequest.Combos).
type ComboItem struct {
	ComboID  string `json:"combo_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=20"`
}

type BookingComboResponse struct {
	ComboID     string     `json:"combo_id"`
	Name        string     `json:"name"`
	Quantity    int        `json:"quantity"`
	Price       int64      `json:"price"`
	Delivered   bool       `json:"delivered"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// SetComboStockRequest caps a combo's stock at one branch (Phần 2.2).
type SetComboStockRequest struct {
	BranchID      string `json:"branch_id" binding:"required"`
	StockQuantity int    `json:"stock_quantity" binding:"min=0"`
}
