package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Lifecycle: PENDING -> CONFIRMED | EXPIRED | REFUNDED.
const (
	BookingPending   = "pending"
	BookingConfirmed = "confirmed"
	BookingExpired   = "expired"
	BookingRefunded  = "refunded"
)

// A booking is sold online (shown in the customer's list) or at the counter for
// a walk-in without an account (no user_id, no payment, no email).
const (
	SoldViaOnline  = "online"
	SoldViaCounter = "counter"
)

const (
	ReasonReplaced        = "replaced" // a newer hold of the same user/show replaced it
	ReasonHoldExpired     = "hold_expired"
	ReasonSeatsLost       = "seats_lost"      // a held seat was swept or taken over
	ReasonShowtimeClosed  = "showtime_closed" // showtime closed or started before confirm
	ReasonAmountMismatch  = "amount_mismatch"
	ReasonPaidAfterExpiry = "paid_after_expiry"
	ReasonCanceled          = "canceled" // the customer released the hold
	ReasonShowtimeCancelled = "showtime_cancelled" // the cinema cancelled the whole showtime
)

// At most one PENDING booking per user per showtime (partial unique index).
type Booking struct {
	ID             string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         string     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	ShowtimeID     string     `gorm:"type:uuid;not null" json:"showtime_id"`
	Status         string     `gorm:"type:varchar(16);not null;default:pending" json:"status"`
	StatusReason   *string    `gorm:"type:varchar(64)" json:"status_reason,omitempty"`
	TotalAmount    int64      `gorm:"not null;default:0" json:"total_amount"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	IdempotencyKey *string    `gorm:"type:varchar(128)" json:"idempotency_key,omitempty"`
	// SoldVia: online bookings carry a user and a payment; counter bookings are
	// walk-in sales with no account, no payment and no ticket email.
	SoldVia        string     `gorm:"type:varchar(16);not null;default:online" json:"sold_via"`
	CustomerName   string     `gorm:"type:varchar(255)" json:"customer_name,omitempty"`
	CustomerPhone  string     `gorm:"type:varchar(20)" json:"customer_phone,omitempty"`
	// PaymentID is the attempt whose collected money this booking carries.
	PaymentID         *string    `gorm:"type:uuid" json:"payment_id,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	FinalizeAttempts  int        `gorm:"not null;default:0" json:"-"`
	NextFinalizeAt    *time.Time `json:"-"`
	EmailSentAt       *time.Time `json:"email_sent_at,omitempty"`
	EmailAttempts     int        `gorm:"not null;default:0" json:"-"`
	EmailClaimedUntil *time.Time `json:"-"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (Booking) TableName() string { return "bookings" }

func (b *Booking) BeforeCreate(*gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

// Snapshot taken at hold time: the price is locked and HoldVersion is the fencing token confirm must match.
type BookingSeat struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID      string    `gorm:"type:uuid;not null" json:"booking_id"`
	ShowtimeSeatID string    `gorm:"type:uuid;not null" json:"showtime_seat_id"`
	SeatType       string    `gorm:"type:varchar(16);not null" json:"seat_type"`
	Price          int64     `gorm:"not null" json:"price"`
	HoldVersion    int64     `gorm:"not null" json:"hold_version"`
	CreatedAt      time.Time `json:"created_at"`
}

func (BookingSeat) TableName() string { return "booking_seats" }

func (s *BookingSeat) BeforeCreate(*gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	return nil
}

const (
	TicketIssued   = "issued"
	TicketRedeemed = "redeemed"
	// TicketVoid marks a ticket whose showtime was cancelled by the cinema; it can
	// never be redeemed again.
	TicketVoid = "void"
)

const (
	RedeemOK        = "ok"
	RedeemUsed      = "used"
	RedeemWrongShow = "wrong_show"
	RedeemNotFound  = "not_found"
	RedeemTooEarly  = "too_early" // before the check-in window opens
	RedeemClosed    = "closed"    // after the check-in window closed
)

type Ticket struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID      string    `gorm:"type:uuid;not null" json:"booking_id"`
	ShowtimeSeatID string    `gorm:"type:uuid;not null" json:"showtime_seat_id"`
	Price          int64     `gorm:"not null" json:"price"`
	Code           string    `gorm:"type:varchar(32);not null" json:"code"`
	Status         string    `gorm:"type:varchar(16);not null;default:issued" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Ticket) TableName() string { return "tickets" }

func (t *Ticket) BeforeCreate(*gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	return nil
}
