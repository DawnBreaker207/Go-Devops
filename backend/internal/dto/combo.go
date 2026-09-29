package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type ComboResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       int64  `json:"price"`
	ImageURL    string `json:"image_url,omitempty"`
	Active      bool   `json:"active"`
}

func NewComboResponse(c *models.Combo) ComboResponse {
	return ComboResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Price:       c.Price,
		ImageURL:    c.ImageURL,
		Active:      c.Active,
	}
}

func NewComboResponses(combos []models.Combo) []ComboResponse {
	out := make([]ComboResponse, 0, len(combos))
	for i := range combos {
		out = append(out, NewComboResponse(&combos[i]))
	}
	return out
}

type ComboOrderItemRequest struct {
	ComboID  string `json:"combo_id" binding:"required,uuid"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=20"`
}

// BookingID is an optional correlation, never required.
type CreateComboOrderRequest struct {
	BookingID string                  `json:"booking_id" binding:"omitempty,uuid"`
	Items     []ComboOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// Walk-in concession sale: no account/booking, handed over at once.
type CounterComboOrderRequest struct {
	Items        []ComboOrderItemRequest `json:"items" binding:"required,min=1,dive"`
	PayMethod    string                  `json:"pay_method" binding:"required,oneof=cash pos"`
	CustomerName string                  `json:"customer_name" binding:"omitempty,max=255"`
}

type ComboOrderItemResponse struct {
	ComboID   string `json:"combo_id"`
	ComboName string `json:"combo_name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Subtotal  int64  `json:"subtotal"`
}

type ComboOrderResponse struct {
	ID           string                   `json:"id"`
	BookingID    string                   `json:"booking_id,omitempty"`
	Status       string                   `json:"status"`
	Total        int64                    `json:"total"`
	SoldChannel  string                   `json:"sold_channel"`
	PayMethod    string                   `json:"pay_method,omitempty"`
	CustomerName string                   `json:"customer_name,omitempty"`
	Items        []ComboOrderItemResponse `json:"items"`
	CreatedAt    time.Time                `json:"created_at"`
}

func NewComboOrderResponse(o *models.ComboOrder, items []models.ComboOrderItem) ComboOrderResponse {
	res := ComboOrderResponse{
		ID:          o.ID,
		Status:      o.Status,
		Total:       o.Total,
		SoldChannel: o.SoldChannel,
		CreatedAt:   o.CreatedAt,
		Items:       make([]ComboOrderItemResponse, 0, len(items)),
	}
	if o.BookingID != nil {
		res.BookingID = *o.BookingID
	}
	if o.PayMethod != nil {
		res.PayMethod = *o.PayMethod
	}
	if o.CustomerName != nil {
		res.CustomerName = *o.CustomerName
	}
	for _, item := range items {
		res.Items = append(res.Items, ComboOrderItemResponse{
			ComboID:   item.ComboID,
			ComboName: item.ComboName,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.UnitPrice * int64(item.Quantity),
		})
	}
	return res
}

// One line on the counter handover board.
type ComboPickupResponse struct {
	OrderID      string                 `json:"order_id"`
	CustomerName string                 `json:"customer_name"`
	CustomerMail string                 `json:"customer_email"`
	BookingID    string                 `json:"booking_id,omitempty"`
	MovieTitle   string                 `json:"movie_title,omitempty"`
	ShowtimeAt   *time.Time             `json:"showtime_at,omitempty"`
	Total        int64                  `json:"total"`
	Items        []ComboOrderItemResponse `json:"items"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Also shows inactive, unlike public GET /combos.
type AdminComboListQuery struct {
	PageQuery
	Active *bool `form:"active"`
}

// Price is whole VND; 0 means a giveaway.
type CreateComboRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=255" example:"Combo 1 - Bap ngot + 2 Pepsi"`
	Description string `json:"description" binding:"omitempty,max=2000"`
	Price       int64  `json:"price" binding:"min=0" example:"89000"`
	ImageURL    string `json:"image_url" binding:"omitempty,url,max=512"`
	// Omitted defaults to TRUE (new products go on sale immediately).
	Active *bool `json:"active"`
}

// Sending no field at all is a 400.
type UpdateComboRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Price       *int64  `json:"price" binding:"omitempty,min=0"`
	ImageURL    *string `json:"image_url" binding:"omitempty,max=512"`
	Active      *bool   `json:"active"`
}

// The service rejects a no-op UPDATE.
func (r UpdateComboRequest) IsEmpty() bool {
	return r.Name == nil && r.Description == nil && r.Price == nil &&
		r.ImageURL == nil && r.Active == nil
}
