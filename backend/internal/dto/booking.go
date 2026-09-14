package dto

import "time"

// HoldRequest reserves seats for a showtime. seat_ids are showtime_seat ids
// from GET /shows/:id/seats.
type HoldRequest struct {
	ShowID         string   `json:"show_id" binding:"required"`
	SeatIDs        []string `json:"seat_ids" binding:"required,min=1"`
	IdempotencyKey string   `json:"idempotency_key" binding:"omitempty,min=1,max=128"`
}

// HeldSeat is one reserved seat with its locked price.
type HeldSeat struct {
	ShowtimeSeatID string `json:"showtime_seat_id"`
	Label          string `json:"label"`
	RowLabel       string `json:"row_label"`
	ColNumber      int    `json:"col_number"`
	SeatType       string `json:"seat_type"`
	Price          int64  `json:"price"`
}

// HoldResponse is the result of POST /orders/hold.
type HoldResponse struct {
	BookingID         string     `json:"booking_id"`
	ShowtimeID        string     `json:"showtime_id"`
	TotalAmount       int64      `json:"total_amount"`
	ExpiresAt         time.Time  `json:"expires_at"`
	Seats             []HeldSeat `json:"seats"`
	ReplacedBookingID string     `json:"replaced_booking_id,omitempty"`
}

// PayRequest opens a checkout with one of the enabled providers (see GET
// /payments/providers); empty uses the configured default. The amount always
// comes from the booking, never from the client (R-P4).
type PayRequest struct {
	Provider string `json:"provider" binding:"omitempty,max=32" example:"mock"`
	ClientIP string `json:"-"`
}

// PayResponse returns the provider checkout to redirect the customer to.
type PayResponse struct {
	PaymentID   string     `json:"payment_id"`
	Provider    string     `json:"provider"`
	TxnRef      string     `json:"txn_ref"`
	RedirectURL string     `json:"redirect_url"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// OrderStatusResponse describes one booking.
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

// OrderShowtime is what the e-ticket shows about the showtime (UC-03).
type OrderShowtime struct {
	MovieID    string    `json:"movie_id"`
	MovieTitle string    `json:"movie_title"`
	HallID     string    `json:"hall_id"`
	HallName   string    `json:"hall_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Started    bool      `json:"started"`
	Ended      bool      `json:"ended"`
}

// TicketResponse is one issued ticket; the frontend renders the QR from code.
type TicketResponse struct {
	ID             string `json:"id"`
	ShowtimeSeatID string `json:"showtime_seat_id"`
	SeatLabel      string `json:"seat_label"`
	SeatType       string `json:"seat_type"`
	Price          int64  `json:"price"`
	Code           string `json:"code"`
	Status         string `json:"status"`
}

// OrderDetailResponse couples a booking with its tickets.
type OrderDetailResponse struct {
	OrderStatusResponse
	Tickets []TicketResponse `json:"tickets"`
}

// RedeemRequestBody is what staff send to check a ticket in at the gate.
type RedeemRequestBody struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
}

// RedeemResponse is the check-in verdict: ok | used | wrong_show | not_found.
type RedeemResponse struct {
	Status     string     `json:"status"`
	TicketID   string     `json:"ticket_id,omitempty"`
	ShowtimeID string     `json:"showtime_id,omitempty"`
	MovieTitle string     `json:"movie_title,omitempty"`
	HallName   string     `json:"hall_name,omitempty"`
	SeatLabel  string     `json:"seat_label,omitempty"`
	StartAt    *time.Time `json:"start_at,omitempty"`
}
