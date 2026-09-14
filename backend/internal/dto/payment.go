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

// PaymentReturnResponse is the result behind a provider return redirect.
type PaymentReturnResponse struct {
	BookingID     string `json:"booking_id"`
	BookingStatus string `json:"booking_status"`
	BookingReason string `json:"booking_reason,omitempty"`
	PaymentID     string `json:"payment_id"`
	PaymentStatus string `json:"payment_status"`
}
