package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Draft; attach codes/combos/articles afterwards.
type CreateCampaignRequest struct {
	Name         string    `json:"name" binding:"required,min=1,max=255" example:"Tet 2027"`
	Description  string    `json:"description" binding:"omitempty,max=2000"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	PerUserLimit *int      `json:"per_user_limit" binding:"omitempty,min=1"`
	// Omitted defaults to FALSE (draft until switched on explicitly).
	Active *bool `json:"active"`
}

type UpdateCampaignRequest struct {
	Name         *string    `json:"name" binding:"omitempty,min=1,max=255"`
	Description  *string    `json:"description" binding:"omitempty,max=2000"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	PerUserLimit *int       `json:"per_user_limit" binding:"omitempty,min=1"`
	Active       *bool      `json:"active"`
}

// The service rejects a no-op UPDATE.
func (r UpdateCampaignRequest) IsEmpty() bool {
	return r.Name == nil && r.Description == nil && r.StartsAt == nil &&
		r.EndsAt == nil && r.PerUserLimit == nil && r.Active == nil
}

type CampaignListQuery struct {
	PageQuery
	Active *bool `form:"active"`
}

type CampaignResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	Active       bool      `json:"active"`
	PerUserLimit int       `json:"per_user_limit"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func NewCampaignResponse(c *models.Campaign) CampaignResponse {
	return CampaignResponse{
		ID:           c.ID,
		Name:         c.Name,
		Description:  c.Description,
		StartsAt:     c.StartsAt,
		EndsAt:       c.EndsAt,
		Active:       c.Active,
		PerUserLimit: c.PerUserLimit,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func NewCampaignResponses(rows []models.Campaign) []CampaignResponse {
	out := make([]CampaignResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewCampaignResponse(&rows[i]))
	}
	return out
}

// PromoPrice is display-only; /combo-orders charges the real price.
type CampaignComboResponse struct {
	ComboID    string `json:"combo_id"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	PromoPrice *int64 `json:"promo_price,omitempty"`
}

type CampaignDiscountCodeResponse = DiscountCodeResponse

type CampaignDetailResponse struct {
	CampaignResponse
	DiscountCodes []CampaignDiscountCodeResponse `json:"discount_codes"`
	Combos        []CampaignComboResponse        `json:"combos"`
	Articles      []ArticleResponse              `json:"articles"`
}

// promo_price is optional, display-only.
type AttachComboRequest struct {
	PromoPrice *int64 `json:"promo_price" binding:"omitempty,min=0"`
}

// CampaignPublicDiscountCode hides the code's rules; checks still run server-side.
type CampaignPublicDiscountCode struct {
	Code string `json:"code"`
	// Nil when the code has no max_uses (unlimited).
	Remaining *int `json:"remaining,omitempty"`
}

// CampaignPublicResponse is only returned while Active && now is inside [starts_at, ends_at).
type CampaignPublicResponse struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	Description   string                       `json:"description,omitempty"`
	StartsAt      time.Time                    `json:"starts_at"`
	EndsAt        time.Time                    `json:"ends_at"`
	DiscountCodes []CampaignPublicDiscountCode `json:"discount_codes"`
	Combos        []CampaignComboResponse      `json:"combos"`
	Articles      []ArticleResponse            `json:"articles"`
}
