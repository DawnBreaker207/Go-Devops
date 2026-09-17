package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	VoucherDiscountFixed      = "fixed"
	VoucherDiscountPercentage = "percentage"
)

const (
	VoucherScopeSeatOnly  = "seat_only"
	VoucherScopeComboOnly = "combo_only"
	VoucherScopeAll       = "all"
)

const (
	VoucherPendingApproval = "pending_approval"
	VoucherActive          = "active"
	VoucherRejected        = "rejected"
	VoucherDisabled        = "disabled"
)

type Voucher struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	Code            string    `gorm:"type:varchar(32);not null" json:"code"`
	DiscountType    string    `gorm:"type:varchar(16);not null" json:"discount_type"`
	DiscountValue   int64     `gorm:"not null" json:"discount_value"`
	MaxDiscount     *int64    `json:"max_discount,omitempty"`
	MinOrderAmount  int64     `gorm:"not null;default:0" json:"min_order_amount"`
	StartsAt        time.Time `gorm:"not null" json:"starts_at"`
	EndsAt          time.Time `gorm:"not null" json:"ends_at"`
	MaxUsage        int       `gorm:"not null" json:"max_usage"`
	UsageCount      int       `gorm:"not null;default:0" json:"usage_count"`
	MaxUsagePerUser int       `gorm:"not null;default:1" json:"max_usage_per_user"`
	ApplyScope      string    `gorm:"type:varchar(16);not null;default:all" json:"apply_scope"`
	BranchID        *string   `gorm:"type:uuid" json:"branch_id,omitempty"`
	CampaignID      *string   `gorm:"type:uuid" json:"campaign_id,omitempty"`
	// CreatedByAdminID is nil for a system-issued loyalty milestone voucher
	// (IsSystemIssued=true); AssignedUserID locks that voucher to the one
	// user who earned it (nil for an ordinary campaign voucher).
	CreatedByAdminID *string   `gorm:"type:uuid" json:"created_by_admin_id,omitempty"`
	IsSystemIssued   bool      `gorm:"not null;default:false" json:"is_system_issued"`
	AssignedUserID   *string   `gorm:"type:uuid" json:"assigned_user_id,omitempty"`
	Status           string    `gorm:"type:varchar(20);not null;default:pending_approval" json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Voucher) TableName() string { return "vouchers" }

func (v *Voucher) BeforeCreate(*gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	return nil
}

// One row per (voucher, booking) — a booking carries at most one voucher.
type VoucherRedemption struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	VoucherID string    `gorm:"type:uuid;not null" json:"voucher_id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
	BookingID string    `gorm:"type:uuid;not null" json:"booking_id"`
	UsedAt    time.Time `json:"used_at"`
}

func (VoucherRedemption) TableName() string { return "voucher_redemptions" }

func (r *VoucherRedemption) BeforeCreate(*gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}
