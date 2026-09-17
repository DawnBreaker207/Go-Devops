package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LedgerEntry unifies every money-like movement into one queryable table
// (ADVANCED_FEATURES_DISCUSSION.md Phần 2.5): a revenue report becomes one
// GROUP BY here instead of a UNION ALL across payments/bookings/
// user_memberships/point_transactions. Written alongside the transaction
// that moves the money, never derived after the fact.
const (
	LedgerPaymentCaptured    = "payment_captured"
	LedgerPaymentRefunded    = "payment_refunded"
	LedgerMembershipPurchase = "membership_purchase"
	LedgerRewardRedeemed     = "reward_redeemed"
)

type LedgerEntry struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	Type           string    `gorm:"type:varchar(32);not null" json:"type"`
	Amount         int64     `gorm:"not null" json:"amount"`
	ReferenceTable string    `gorm:"type:varchar(32);not null" json:"reference_table"`
	ReferenceID    string    `gorm:"type:uuid;not null" json:"reference_id"`
	UserID         *string   `gorm:"type:uuid" json:"user_id,omitempty"`
	BookingID      *string   `gorm:"type:uuid" json:"booking_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (LedgerEntry) TableName() string { return "ledger_entries" }

func (e *LedgerEntry) BeforeCreate(*gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
