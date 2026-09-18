package dto

import "time"

// StaffShowtimeResponse is one showtime on the staff board. It carries seat
// counts only: staff never see money
type StaffShowtimeResponse struct {
	ID         string    `json:"id"`
	MovieTitle string    `json:"movie_title"`
	HallName   string    `json:"hall_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Status     string    `json:"status"`
	Capacity   int       `json:"capacity"`
	Held       int       `json:"held"`
	Sold       int       `json:"sold"`
	Available  int       `json:"available"`
	CheckedIn  int       `json:"checked_in"`
}

// StaffBoardResponse is the staff dashboard of one day.
type StaffBoardResponse struct {
	Date      string                  `json:"date" example:"2026-09-14"`
	Showtimes []StaffShowtimeResponse `json:"showtimes"`
}

// BoxOfficeDayResponse settles the counter's day: walk-in sales only.
type BoxOfficeDayResponse struct {
	Date  string `json:"date" example:"2026-09-14"`
	Count int64  `json:"count"`
	Total int64  `json:"total"`
}

// StaffTicketResponse is one ticket of a showtime at the gate.
type StaffTicketResponse struct {
	ID        string    `json:"id"`
	BookingID string    `json:"booking_id"`
	SeatLabel string    `json:"seat_label"`
	SeatType  string    `json:"seat_type"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyReportResponse is the admin revenue report over closed days.
type DailyReportResponse struct {
	From         string                   `json:"from" example:"2026-09-08"`
	To           string                   `json:"to" example:"2026-09-14"`
	TotalRevenue int64                    `json:"total_revenue"`
	TicketsSold  int                      `json:"tickets_sold"`
	Days         []DailyAggregateResponse `json:"days"`
}

// DailyAggregateResponse is the closeDay rollup of one day.
type DailyAggregateResponse struct {
	ReportDate    string         `json:"report_date"`
	TotalRevenue  int64          `json:"total_revenue"`
	TicketsSold   int            `json:"tickets_sold"`
	SeatsSold     int            `json:"seats_sold"`
	Capacity      int            `json:"capacity"`
	OccupancyRate float64        `json:"occupancy_rate"`
	Breakdown     map[string]any `json:"breakdown"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// StuckRefundAlert is a refund that has failed provider-side several times in a row.
type StuckRefundAlert struct {
	PaymentID string `json:"payment_id"`
	BookingID string `json:"booking_id"`
	Attempts  int    `json:"attempts"`
	Amount    int64  `json:"amount"`
	LastError string `json:"last_error,omitempty"`
}

// FailedJobAlert is a recent batch job run that ended in failure.
type FailedJobAlert struct {
	ID           string    `json:"id"`
	JobName      string    `json:"job_name"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
}

// GivenUpEmailAlert is a confirmed order whose ticket email exhausted every retry.
type GivenUpEmailAlert struct {
	BookingID string    `json:"booking_id"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminAlertsResponse surfaces operational issues that would otherwise only
// be found by manually filtering /admin/audit-logs or /admin/batch/jobs.
type AdminAlertsResponse struct {
	StuckRefunds  []StuckRefundAlert  `json:"stuck_refunds"`
	FailedJobs    []FailedJobAlert    `json:"failed_jobs"`
	GivenUpEmails []GivenUpEmailAlert `json:"given_up_emails"`
}

// AdminOverviewResponse is the one-call admin dashboard: today's business
// computed live (not waiting on closeDay), the last 7 closed days for trend,
// what's still to come today, and anything that needs an admin's attention.
type AdminOverviewResponse struct {
	Today             DailyAggregateResponse   `json:"today"`
	Last7Days         []DailyAggregateResponse `json:"last_7_days"`
	UpcomingShowtimes []StaffShowtimeResponse  `json:"upcoming_showtimes"`
	Alerts            AdminAlertsResponse      `json:"alerts"`
}

// StaffOverviewResponse composes the existing staff board and box office
// numbers with one derived count, so the floor app can land on a single call.
type StaffOverviewResponse struct {
	Date              string                  `json:"date" example:"2026-09-14"`
	Showtimes         []StaffShowtimeResponse `json:"showtimes"`
	CounterSalesCount int64                   `json:"counter_sales_count"`
	CounterSalesTotal int64                   `json:"counter_sales_total"`
	AwaitingCheckin   int                     `json:"awaiting_checkin"`
}

// AdminStatsResponse is the admin dashboard's four headline counts, one per
// tile. Soft-deleted movies, showtimes and users are excluded; bookings counts
// CONFIRMED orders only, so a hold nobody paid for never inflates a tile
// labelled "Bookings". Locked (active=false) users are still counted: a lock is
// an operational state, not a deletion — and GET /admin/users counts them too.
type AdminStatsResponse struct {
	Movies    int64 `json:"movies"`
	Showtimes int64 `json:"showtimes"`
	Bookings  int64 `json:"bookings"`
	Users     int64 `json:"users"`
}
