package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PointEarn           = "earn"
	PointRedeem         = "redeem"
	PointRefundReversal = "refund_reversal"
)

const (
	RewardGift    = "gift"
	RewardVoucher = "voucher"
)

const (
	RewardRedemptionPendingPickup = "pending_pickup"
	RewardRedemptionDelivered     = "delivered"
)

// UserPoints is the atomic source of the live balance; never derive it by
// summing PointTransaction (Phần 2.1 — avoids a TOCTOU race between two
// concurrent redeems).
type UserPoints struct {
	UserID                 string    `gorm:"type:uuid;primaryKey" json:"user_id"`
	Balance                int64     `gorm:"not null;default:0" json:"balance"`
	TicketsTowardMilestone int       `gorm:"not null;default:0" json:"-"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (UserPoints) TableName() string { return "user_points" }

// PointTransaction is an append-only log/audit trail, not a computation source.
type PointTransaction struct {
	ID                 string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID             string    `gorm:"type:uuid;not null" json:"user_id"`
	Amount             int64     `gorm:"not null" json:"amount"`
	Type               string    `gorm:"type:varchar(16);not null" json:"type"`
	ReferenceBookingID *string   `gorm:"type:uuid" json:"reference_booking_id,omitempty"`
	ReferenceRewardID  *string   `gorm:"type:uuid" json:"reference_reward_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

func (PointTransaction) TableName() string { return "point_transactions" }

func (t *PointTransaction) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}

type Reward struct {
	ID                string    `gorm:"type:uuid;primaryKey" json:"id"`
	Type              string    `gorm:"type:varchar(16);not null" json:"type"`
	Name              string    `gorm:"type:varchar(255);not null" json:"name"`
	PointsCost        int64     `gorm:"not null" json:"points_cost"`
	StockQuantity     *int      `json:"stock_quantity,omitempty"`
	VoucherTemplateID *string   `gorm:"type:uuid" json:"voucher_template_id,omitempty"`
	Active            bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Reward) TableName() string { return "rewards" }

func (r *Reward) BeforeCreate(*gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RewardRedemption: a gift is picked up at the counter (status starts
// pending_pickup, staff marks delivered by code, same pattern as
// booking_combos); a voucher reward is delivered digitally at once.
type RewardRedemption struct {
	ID              string     `gorm:"type:uuid;primaryKey" json:"id"`
	RewardID        string     `gorm:"type:uuid;not null" json:"reward_id"`
	UserID          string     `gorm:"type:uuid;not null" json:"user_id"`
	PointsSpent     int64      `gorm:"not null" json:"points_spent"`
	Code            string     `gorm:"type:varchar(32);not null" json:"code"`
	Status          string     `gorm:"type:varchar(16);not null;default:pending_pickup" json:"status"`
	IssuedVoucherID *string    `gorm:"type:uuid" json:"issued_voucher_id,omitempty"`
	DeliveredAt     *time.Time `json:"delivered_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (RewardRedemption) TableName() string { return "reward_redemptions" }

func (r *RewardRedemption) BeforeCreate(*gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}
