package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// VoucherRequest creates a voucher in status=pending_approval (Phần 9.3
// maker-checker groundwork: creation and approval are separate actions).
type VoucherRequest struct {
	Code            string  `json:"code" binding:"required,min=3,max=32" example:"SALE50"`
	DiscountType    string  `json:"discount_type" binding:"required,oneof=fixed percentage" example:"percentage"`
	DiscountValue   int64   `json:"discount_value" binding:"required,min=1" example:"50000"`
	MaxDiscount     *int64  `json:"max_discount" binding:"omitempty,min=1"`
	MinOrderAmount  int64   `json:"min_order_amount" binding:"omitempty,min=0"`
	StartsAt        time.Time `json:"starts_at" binding:"required"`
	EndsAt          time.Time `json:"ends_at" binding:"required"`
	MaxUsage        int     `json:"max_usage" binding:"required,min=1"`
	MaxUsagePerUser int     `json:"max_usage_per_user" binding:"omitempty,min=1"`
	ApplyScope      string  `json:"apply_scope" binding:"omitempty,oneof=seat_only combo_only all" example:"all"`
}

type UpdateVoucherRequest struct {
	DiscountType    *string    `json:"discount_type" binding:"omitempty,oneof=fixed percentage"`
	DiscountValue   *int64     `json:"discount_value" binding:"omitempty,min=1"`
	MaxDiscount     *int64     `json:"max_discount" binding:"omitempty,min=1"`
	MinOrderAmount  *int64     `json:"min_order_amount" binding:"omitempty,min=0"`
	StartsAt        *time.Time `json:"starts_at"`
	EndsAt          *time.Time `json:"ends_at"`
	MaxUsage        *int       `json:"max_usage" binding:"omitempty,min=1"`
	MaxUsagePerUser *int       `json:"max_usage_per_user" binding:"omitempty,min=1"`
	ApplyScope      *string    `json:"apply_scope" binding:"omitempty,oneof=seat_only combo_only all"`
}

// VoucherStatusRequest moves a voucher between pending_approval/active/rejected/disabled.
type VoucherStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active rejected disabled" example:"active"`
}

type VoucherResponse struct {
	ID                string    `json:"id"`
	Code              string    `json:"code"`
	DiscountType      string    `json:"discount_type"`
	DiscountValue     int64     `json:"discount_value"`
	MaxDiscount       *int64    `json:"max_discount,omitempty"`
	MinOrderAmount    int64     `json:"min_order_amount"`
	StartsAt          time.Time `json:"starts_at"`
	EndsAt            time.Time `json:"ends_at"`
	MaxUsage          int       `json:"max_usage"`
	UsageCount        int       `json:"usage_count"`
	MaxUsagePerUser   int       `json:"max_usage_per_user"`
	ApplyScope        string    `json:"apply_scope"`
	CreatedByAdminID  *string   `json:"created_by_admin_id,omitempty"`
	IsSystemIssued    bool      `json:"is_system_issued"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func NewVoucherResponse(v *models.Voucher) VoucherResponse {
	return VoucherResponse{
		ID: v.ID, Code: v.Code, DiscountType: v.DiscountType, DiscountValue: v.DiscountValue,
		MaxDiscount: v.MaxDiscount, MinOrderAmount: v.MinOrderAmount, StartsAt: v.StartsAt, EndsAt: v.EndsAt,
		MaxUsage: v.MaxUsage, UsageCount: v.UsageCount, MaxUsagePerUser: v.MaxUsagePerUser,
		ApplyScope: v.ApplyScope, CreatedByAdminID: v.CreatedByAdminID, IsSystemIssued: v.IsSystemIssued, Status: v.Status,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

func NewVoucherResponses(rows []models.Voucher) []VoucherResponse {
	out := make([]VoucherResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewVoucherResponse(&rows[i]))
	}
	return out
}

// VoucherConflictResponse: a redemption where the redeeming user is also the
// voucher's own creator (Phần 9.4).
type VoucherConflictResponse struct {
	VoucherID string    `json:"voucher_id"`
	Code      string    `json:"code"`
	AdminID   string    `json:"admin_id"`
	BookingID string    `json:"booking_id"`
	UsedAt    time.Time `json:"used_at"`
}
