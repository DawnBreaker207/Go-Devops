package dto

import "time"

// seat_ids are showtime_seat ids from GET /shows/:id/seats.
type HoldRequest struct {
	ShowID         string   `json:"show_id" binding:"required"`
	SeatIDs        []string `json:"seat_ids" binding:"required,min=1"`
	IdempotencyKey string   `json:"idempotency_key" binding:"omitempty,min=1,max=128"`
}

// Walk-in sale: cash, confirmed immediately, no account/provider.
type CounterSellRequest struct {
	ShowID        string   `json:"show_id" binding:"required"`
	SeatIDs       []string `json:"seat_ids" binding:"required,min=1"`
	CustomerName  string   `json:"customer_name" binding:"omitempty,max=255"`
	CustomerPhone string   `json:"customer_phone" binding:"omitempty,max=20"`
}

// Base64 PNG, same encoding as ticket emails.
type TicketQRResponse struct {
	TicketID string `json:"ticket_id"`
	Code     string `json:"code"`
	QRBase64 string `json:"qr_base64"`
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

// Seatless PENDING booking; seats attach later via POST /orders/hold.
type InitRequest struct {
	ShowID string `json:"show_id" binding:"required"`
}

type InitResponse struct {
	BookingID  string    `json:"booking_id"`
	ShowtimeID string    `json:"showtime_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int64 `json:"ttl_seconds"`
	Reused     bool  `json:"reused"`
}

type RefreshResponse struct {
	BookingID string    `json:"booking_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Empty provider uses the default; amount always comes from the booking.
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
	ID           string `json:"id"`
	ShowtimeID   string `json:"showtime_id"`
	Status       string `json:"status"`
	StatusReason string `json:"status_reason,omitempty"`
	// TotalAmount is the SEAT SUBTOTAL; PayableAmount is what the customer actually owes.
	TotalAmount    int64           `json:"total_amount"`
	DiscountAmount int64           `json:"discount_amount"`
	PayableAmount  int64           `json:"payable_amount"`
	CreatedAt      time.Time       `json:"created_at"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	PaidAt         *time.Time      `json:"paid_at,omitempty"`
	CollectedAt    *time.Time      `json:"collected_at,omitempty"`
	Payment        *PaymentSummary `json:"payment,omitempty"`
	Showtime       *OrderShowtime  `json:"showtime,omitempty"`
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

type RedeemResponse struct {
	Status     string     `json:"status"`
	TicketID   string     `json:"ticket_id,omitempty"`
	ShowtimeID string     `json:"showtime_id,omitempty"`
	MovieTitle string     `json:"movie_title,omitempty"`
	AgeRating  string     `json:"age_rating,omitempty"`
	HallName   string     `json:"hall_name,omitempty"`
	SeatLabel  string     `json:"seat_label,omitempty"`
	StartAt    *time.Time `json:"start_at,omitempty"`
	CheckinOpensAt  *time.Time `json:"checkin_opens_at,omitempty"`
	CheckinClosesAt *time.Time `json:"checkin_closes_at,omitempty"`
}

// Unscoped: online + counter side by side.
// date/from/to bound created_at as local days, same as ShowtimeAdminListQuery.
type AdminOrderListQuery struct {
	PageQuery
	Status string `form:"status" binding:"omitempty,oneof=pending confirmed expired refunded"`
	// PaymentStatus matches the carried attempt (bookings.payment_id); an unpaid open checkout carries none.
	PaymentStatus string `form:"payment_status" binding:"omitempty,oneof=pending paid failed refund_pending refunded"`
	SoldVia       string `form:"sold_via" binding:"omitempty,oneof=online counter"`
	ShowtimeID    string `form:"showtime_id" binding:"omitempty,uuid"`
	MovieID       string `form:"movie_id" binding:"omitempty,uuid"`
	UserID        string `form:"user_id" binding:"omitempty,uuid"`
	// Narrows to one calendar day, wins over From/To.
	Date  string `form:"date" binding:"omitempty,datetime=2006-01-02"`
	From  string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To    string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	Sort  string `form:"sort" binding:"omitempty,oneof=created_at paid_at total_amount start_at"`
	Order string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// Online carries the account; counter only till name/phone.
type OrderCustomer struct {
	UserID   string `json:"user_id,omitempty"`
	Email    string `json:"email,omitempty"`
	FullName string `json:"full_name,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

type AdminOrderListItem struct {
	OrderStatusResponse
	SoldVia  string         `json:"sold_via"`
	Seats    int            `json:"seats"`
	Customer *OrderCustomer `json:"customer,omitempty"`
}
