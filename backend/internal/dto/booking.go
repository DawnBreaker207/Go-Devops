package dto

import "time"

// seat_ids are showtime_seat ids from GET /shows/:id/seats.
type HoldRequest struct {
	ShowID         string   `json:"show_id" binding:"required"`
	SeatIDs        []string `json:"seat_ids" binding:"required,min=1"`
	IdempotencyKey string   `json:"idempotency_key" binding:"omitempty,min=1,max=128"`
}

// CounterSellRequest is a walk-in sale at the till: no account, no payment
// provider; cash is collected and the booking is confirmed immediately.
type CounterSellRequest struct {
	ShowID        string   `json:"show_id" binding:"required"`
	SeatIDs       []string `json:"seat_ids" binding:"required,min=1"`
	CustomerName  string   `json:"customer_name" binding:"omitempty,max=255"`
	CustomerPhone string   `json:"customer_phone" binding:"omitempty,max=20"`
}

type HeldSeat struct {
	ShowtimeSeatID string `json:"showtime_seat_id"`
	Label          string `json:"label"`
	RowLabel       string `json:"row_label"`
	ColNumber      int    `json:"col_number"`
	SeatType       string `json:"seat_type"`
	Price          int64  `json:"price"`
}

type HoldResponse struct {
	BookingID         string     `json:"booking_id"`
	ShowtimeID        string     `json:"showtime_id"`
	TotalAmount       int64      `json:"total_amount"`
	ExpiresAt         time.Time  `json:"expires_at"`
	Seats             []HeldSeat `json:"seats"`
	ReplacedBookingID string     `json:"replaced_booking_id,omitempty"`
}

// An empty provider uses the configured default (see GET /payments/providers).
// The amount always comes from the booking, never from the client.
type PayRequest struct {
	Provider string `json:"provider" binding:"omitempty,max=32" example:"mock"`
	ClientIP string `json:"-"`
}

type PayResponse struct {
	PaymentID   string     `json:"payment_id"`
	Provider    string     `json:"provider"`
	TxnRef      string     `json:"txn_ref"`
	RedirectURL string     `json:"redirect_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type OrderStatusResponse struct {
	ID           string          `json:"id"`
	ShowtimeID   string          `json:"showtime_id"`
	Status       string          `json:"status"`
	StatusReason string          `json:"status_reason,omitempty"`
	TotalAmount  int64           `json:"total_amount"`
	CreatedAt    time.Time       `json:"created_at"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	PaidAt       *time.Time      `json:"paid_at,omitempty"`
	Payment      *PaymentSummary `json:"payment,omitempty"`
	Showtime     *OrderShowtime  `json:"showtime,omitempty"`
}

type OrderShowtime struct {
	MovieID    string    `json:"movie_id"`
	MovieTitle string    `json:"movie_title"`
	AgeRating  string    `json:"age_rating"`
	HallID     string    `json:"hall_id"`
	HallName   string    `json:"hall_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Started    bool      `json:"started"`
	Ended      bool      `json:"ended"`
}

// The frontend renders the QR from code.
type TicketResponse struct {
	ID             string `json:"id"`
	ShowtimeSeatID string `json:"showtime_seat_id"`
	SeatLabel      string `json:"seat_label"`
	SeatType       string `json:"seat_type"`
	Price          int64  `json:"price"`
	Code           string `json:"code"`
	Status         string `json:"status"`
}

type OrderDetailResponse struct {
	OrderStatusResponse
	Tickets []TicketResponse `json:"tickets"`
}

type RedeemRequestBody struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
}

// Status: ok | used | wrong_show | not_found | too_early | closed.
type RedeemResponse struct {
	Status     string     `json:"status"`
	TicketID   string     `json:"ticket_id,omitempty"`
	ShowtimeID string     `json:"showtime_id,omitempty"`
	MovieTitle string     `json:"movie_title,omitempty"`
	AgeRating  string     `json:"age_rating,omitempty"`
	HallName   string     `json:"hall_name,omitempty"`
	SeatLabel  string     `json:"seat_label,omitempty"`
	StartAt    *time.Time `json:"start_at,omitempty"`
	// Check-in window, so the gate can tell when the doors open or closed.
	CheckinOpensAt  *time.Time `json:"checkin_opens_at,omitempty"`
	CheckinClosesAt *time.Time `json:"checkin_closes_at,omitempty"`
}
