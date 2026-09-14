package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Booking status. Lifecycle: PENDING -> CONFIRMED | EXPIRED | REFUNDED.
const (
	BookingPending   = "pending"
	BookingConfirmed = "confirmed"
	BookingExpired   = "expired"
	BookingRefunded  = "refunded"
)

// Reasons stored in bookings.status_reason when a booking leaves PENDING
// without being confirmed.
const (
	ReasonReplaced        = "replaced"          // a newer hold of the same user/show replaced it (E-HO12)
	ReasonHoldExpired     = "hold_expired"      // TTL passed before payment/confirm (E-C1, E-C4)
	ReasonSeatsLost       = "seats_lost"        // a held seat was swept or taken over (E-C2)
	ReasonShowtimeClosed  = "showtime_closed"   // showtime closed or started before confirm (E-C5)
	ReasonAmountMismatch  = "amount_mismatch"   // provider settled a different amount (E-P4)
	ReasonPaidAfterExpiry = "paid_after_expiry" // payment arrived for an expired booking (E-P3)
	ReasonCanceled        = "canceled"          // the customer released the hold (E-R2)
)

// Booking holds seats for a user before payment. At most one PENDING booking
// per user per showtime (enforced by partial unique index at the DB level).
type Booking struct {
	ID             string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         string     `gorm:"type:uuid;not null" json:"user_id"`
	ShowtimeID     string     `gorm:"type:uuid;not null" json:"showtime_id"`
	Status         string     `gorm:"type:varchar(16);not null;default:pending" json:"status"`
	StatusReason   *string    `gorm:"type:varchar(64)" json:"status_reason,omitempty"`
	TotalAmount    int64      `gorm:"not null;default:0" json:"total_amount"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	IdempotencyKey *string    `gorm:"type:varchar(128)" json:"idempotency_key,omitempty"`
	// PaymentID is the attempt whose collected money this booking carries.
	PaymentID   *string        `gorm:"type:uuid" json:"payment_id,omitempty"`
	PaidAt      *time.Time     `json:"paid_at,omitempty"`
	EmailSentAt *time.Time     `json:"email_sent_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Booking) TableName() string { return "bookings" }

func (b *Booking) BeforeCreate(*gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

// BookingSeat is the seat snapshot of a booking taken at hold time: the price
// is locked (E-HO9) and HoldVersion is the fencing token confirm must match.
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

// Ticket status.
const (
	TicketIssued   = "issued"
	TicketRedeemed = "redeemed"
)

// Redeem outcomes returned by POST /tickets/:id/redeem.
const (
	RedeemOK        = "ok"
	RedeemUsed      = "used"
	RedeemWrongShow = "wrong_show"
	RedeemNotFound  = "not_found"
)

// Ticket represents one seat sold within a booking.
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
