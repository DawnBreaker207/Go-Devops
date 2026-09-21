package dto

import "time"

type PaymentProviderResponse struct {
	Name        string `json:"name" example:"mock"`
	DisplayName string `json:"display_name"`
	Default     bool   `json:"default"`
}

// The attempt whose money the booking carries, else the latest attempt.
type PaymentSummary struct {
	ID           string     `json:"id"`
	Provider     string     `json:"provider"`
	TxnRef       string     `json:"txn_ref"`
	Status       string     `json:"status"`
	StatusReason string     `json:"status_reason,omitempty"`
	Amount       int64      `json:"amount"`
	PaidAmount   *int64     `json:"paid_amount,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty"`
}

// TransactionResponse is one row of the financial view GET
// /users/me/transactions: what was paid/refunded, when, through which
// provider, plus just enough booking/showtime context to place it. This is
// distinct from GET /orders, which is the booking/ticket view.
type TransactionResponse struct {
	PaymentID    string     `json:"payment_id"`
	Provider     string     `json:"provider"`
	TxnRef       string     `json:"txn_ref"`
	Status       string     `json:"status"`
	StatusReason string     `json:"status_reason,omitempty"`
	Amount       int64      `json:"amount"`
	PaidAmount   *int64     `json:"paid_amount,omitempty"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	BookingID    string     `json:"booking_id"`
	ShowtimeID   string     `json:"showtime_id"`
	MovieTitle   string     `json:"movie_title"`
	HallName     string     `json:"hall_name"`
	StartAt      time.Time  `json:"start_at"`
}

// PaymentReturnResponse is the result behind a provider return redirect.
type PaymentReturnResponse struct {
	BookingID     string `json:"booking_id"`
	BookingStatus string `json:"booking_status"`
	BookingReason string `json:"booking_reason,omitempty"`
	PaymentID     string `json:"payment_id"`
	PaymentStatus string `json:"payment_status"`
}
