package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Payment attempt status.
const (
	PaymentPending       = "pending"        // checkout opened, money not confirmed
	PaymentPaid          = "paid"           // money collected and carried by the booking
	PaymentFailed        = "failed"         // declined, canceled, abandoned or never created
	PaymentRefundPending = "refund_pending" // collected but must go back; provider has not acknowledged yet
	PaymentRefunded      = "refunded"
)

// Reasons stored in payments.status_reason.
const (
	PaymentReasonCreateFailed = "create_failed"     // provider refused to open the checkout
	PaymentReasonDeclined     = "declined"          // provider reported a failed payment
	PaymentReasonAbandoned    = "abandoned"         // still unpaid long after the hold ended
	PaymentReasonDuplicate    = "duplicate_payment" // booking already carried another attempt's money
)

// Payment is one payment attempt of a booking with one provider. Mock and
// real gateways use the same row shape.
type Payment struct {
	ID            string            `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID     string            `gorm:"type:uuid;not null" json:"booking_id"`
	Provider      string            `gorm:"type:varchar(32);not null" json:"provider"`
	TxnRef        string            `gorm:"type:varchar(64);not null" json:"txn_ref"`
	Amount        int64             `gorm:"not null" json:"amount"`
	Status        string            `gorm:"type:varchar(16);not null;default:pending" json:"status"`
	StatusReason  *string           `gorm:"type:varchar(64)" json:"status_reason,omitempty"`
	RedirectURL   *string           `json:"redirect_url,omitempty"`
	ProviderTxnID *string           `gorm:"type:varchar(128)" json:"provider_txn_id,omitempty"`
	ProviderData  map[string]string `gorm:"serializer:json;type:jsonb;not null" json:"-"`
	PaidAmount    *int64            `json:"paid_amount,omitempty"`
	PaidAt        *time.Time        `json:"paid_at,omitempty"`
	RefundedAt    *time.Time        `json:"refunded_at,omitempty"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	CheckedAt     *time.Time        `json:"checked_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func (Payment) TableName() string { return "payments" }

func (p *Payment) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.ProviderData == nil {
		p.ProviderData = map[string]string{}
	}
	return nil
}
