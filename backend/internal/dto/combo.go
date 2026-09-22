package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// ComboResponse is one concession/combo product in the public catalog.
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

// ComboOrderItemRequest is one line of a combo order.
type ComboOrderItemRequest struct {
	ComboID  string `json:"combo_id" binding:"required,uuid"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=20"`
}

// CreateComboOrderRequest places a combo/concession order. BookingID is an
// optional correlation to a ticket order (e.g. "pick up with your tickets");
// it is never required and a combo order never touches the booking itself.
type CreateComboOrderRequest struct {
	BookingID string                  `json:"booking_id" binding:"omitempty,uuid"`
	Items     []ComboOrderItemRequest `json:"items" binding:"required,min=1,dive"`
}

// ComboOrderItemResponse is one priced, named line of a placed order.
type ComboOrderItemResponse struct {
	ComboID   string `json:"combo_id"`
	ComboName string `json:"combo_name"`
	Quantity  int    `json:"quantity"`
	UnitPrice int64  `json:"unit_price"`
	Subtotal  int64  `json:"subtotal"`
}

// ComboOrderResponse is a placed combo order with its priced items.
type ComboOrderResponse struct {
	ID        string                   `json:"id"`
	BookingID string                   `json:"booking_id,omitempty"`
	Status    string                   `json:"status"`
	Total     int64                    `json:"total"`
	Items     []ComboOrderItemResponse `json:"items"`
	CreatedAt time.Time                `json:"created_at"`
}

func NewComboOrderResponse(o *models.ComboOrder, items []models.ComboOrderItem) ComboOrderResponse {
	res := ComboOrderResponse{
		ID:        o.ID,
		Status:    o.Status,
		Total:     o.Total,
		CreatedAt: o.CreatedAt,
		Items:     make([]ComboOrderItemResponse, 0, len(items)),
	}
	if o.BookingID != nil {
		res.BookingID = *o.BookingID
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

/* -------------------------------------------------------------------------- */
/* Admin catalogue (operator: admin AND staff, like halls/showtimes)           */
/* -------------------------------------------------------------------------- */

// AdminComboListQuery pages the catalogue an operator manages. Unlike the public
// GET /combos it also shows inactive and lets the caller filter on that, because
// the whole point of the screen is to take a product off sale and put it back.
type AdminComboListQuery struct {
	PageQuery
	// Active is a pointer: absent means "no filter", which is different from false.
	Active *bool `form:"active"`
}

// CreateComboRequest adds a product to the catalogue. Price is int64 whole VND
// like every other amount in this API; 0 is allowed and means a giveaway, not
// "unset" (there is no nullable price on this table).
type CreateComboRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=255" example:"Combo 1 - Bap ngot + 2 Pepsi"`
	Description string `json:"description" binding:"omitempty,max=2000"`
	Price       int64  `json:"price" binding:"min=0" example:"89000"`
	ImageURL    string `json:"image_url" binding:"omitempty,url,max=512"`
	// Active defaults to TRUE when omitted: a product created from the admin
	// screen is normally meant to go on sale immediately.
	Active *bool `json:"active"`
}

// UpdateComboRequest is a PARTIAL update: every field is a pointer and an absent
// one is left alone. Sending no field at all is a 400, matching PATCH
// /admin/users/:id rather than the full-replace PUT the movie/showtime screens use.
type UpdateComboRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Price       *int64  `json:"price" binding:"omitempty,min=0"`
	ImageURL    *string `json:"image_url" binding:"omitempty,max=512"`
	Active      *bool   `json:"active"`
}

// IsEmpty reports a body that would change nothing, which the service rejects
// instead of issuing a no-op UPDATE.
func (r UpdateComboRequest) IsEmpty() bool {
	return r.Name == nil && r.Description == nil && r.Price == nil &&
		r.ImageURL == nil && r.Active == nil
}
