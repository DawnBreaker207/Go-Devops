package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type PointsBalanceResponse struct {
	Balance int64 `json:"balance"`
}

type PointTransactionResponse struct {
	ID                 string    `json:"id"`
	Amount             int64     `json:"amount"`
	Type               string    `json:"type"`
	ReferenceBookingID string    `json:"reference_booking_id,omitempty"`
	ReferenceRewardID  string    `json:"reference_reward_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func NewPointTransactionResponse(t models.PointTransaction) PointTransactionResponse {
	r := PointTransactionResponse{ID: t.ID, Amount: t.Amount, Type: t.Type, CreatedAt: t.CreatedAt}
	if t.ReferenceBookingID != nil {
		r.ReferenceBookingID = *t.ReferenceBookingID
	}
	if t.ReferenceRewardID != nil {
		r.ReferenceRewardID = *t.ReferenceRewardID
	}
	return r
}

type RewardRequest struct {
	Type              string `json:"type" binding:"required,oneof=gift voucher" example:"gift"`
	Name              string `json:"name" binding:"required,min=1,max=255"`
	PointsCost        int64  `json:"points_cost" binding:"required,min=1"`
	StockQuantity     *int   `json:"stock_quantity" binding:"omitempty,min=0"`
	VoucherTemplateID string `json:"voucher_template_id" binding:"omitempty"`
}

type UpdateRewardRequest struct {
	Name              *string `json:"name" binding:"omitempty,min=1,max=255"`
	PointsCost        *int64  `json:"points_cost" binding:"omitempty,min=1"`
	StockQuantity     *int    `json:"stock_quantity" binding:"omitempty,min=0"`
	VoucherTemplateID *string `json:"voucher_template_id"`
	Active            *bool   `json:"active"`
}

type RewardResponse struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	Name              string    `json:"name"`
	PointsCost        int64     `json:"points_cost"`
	StockQuantity     *int      `json:"stock_quantity,omitempty"`
	VoucherTemplateID string    `json:"voucher_template_id,omitempty"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewRewardResponse(r *models.Reward) RewardResponse {
	res := RewardResponse{
		ID: r.ID, Type: r.Type, Name: r.Name, PointsCost: r.PointsCost,
		StockQuantity: r.StockQuantity, Active: r.Active, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.VoucherTemplateID != nil {
		res.VoucherTemplateID = *r.VoucherTemplateID
	}
	return res
}

func NewRewardResponses(rows []models.Reward) []RewardResponse {
	out := make([]RewardResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewRewardResponse(&rows[i]))
	}
	return out
}

type RewardRedemptionResponse struct {
	ID          string     `json:"id"`
	RewardID    string     `json:"reward_id"`
	PointsSpent int64      `json:"points_spent"`
	Code        string     `json:"code"`
	Status      string     `json:"status"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewRewardRedemptionResponse(r *models.RewardRedemption) RewardRedemptionResponse {
	return RewardRedemptionResponse{
		ID: r.ID, RewardID: r.RewardID, PointsSpent: r.PointsSpent, Code: r.Code,
		Status: r.Status, DeliveredAt: r.DeliveredAt, CreatedAt: r.CreatedAt,
	}
}
