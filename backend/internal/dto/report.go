package dto

import "time"

// StaffShowtimeResponse: one showtime, seat counts only (no money).
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

type StaffBoardResponse struct {
	Date      string                  `json:"date" example:"2026-09-14"`
	Showtimes []StaffShowtimeResponse `json:"showtimes"`
}

// BoxOfficeDayResponse: walk-in sales of one day.
type BoxOfficeDayResponse struct {
	Date  string `json:"date" example:"2026-09-14"`
	Count int64  `json:"count"`
	Total int64  `json:"total"`
}

type StaffTicketResponse struct {
	ID        string    `json:"id"`
	BookingID string    `json:"booking_id"`
	SeatLabel string    `json:"seat_label"`
	SeatType  string    `json:"seat_type"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyReportResponse: admin revenue over closed days.
type DailyReportResponse struct {
	From         string                   `json:"from" example:"2026-09-08"`
	To           string                   `json:"to" example:"2026-09-14"`
	TotalRevenue int64                    `json:"total_revenue"`
	TicketsSold  int                      `json:"tickets_sold"`
	Days         []DailyAggregateResponse `json:"days"`
}

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

// StuckRefundAlert: refund failing provider-side repeatedly.
type StuckRefundAlert struct {
	PaymentID string `json:"payment_id"`
	BookingID string `json:"booking_id"`
	Attempts  int    `json:"attempts"`
	Amount    int64  `json:"amount"`
	LastError string `json:"last_error,omitempty"`
}

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

// AdminAlertsResponse: issues otherwise found only by filtering audit-logs/batch-jobs.
type AdminAlertsResponse struct {
	StuckRefunds  []StuckRefundAlert  `json:"stuck_refunds"`
	FailedJobs    []FailedJobAlert    `json:"failed_jobs"`
	GivenUpEmails []GivenUpEmailAlert `json:"given_up_emails"`
}

// AdminOverviewResponse: one-call dashboard (live today, last 7 closed days, upcoming, alerts).
type AdminOverviewResponse struct {
	Today             DailyAggregateResponse   `json:"today"`
	Last7Days         []DailyAggregateResponse `json:"last_7_days"`
	UpcomingShowtimes []StaffShowtimeResponse  `json:"upcoming_showtimes"`
	Alerts            AdminAlertsResponse      `json:"alerts"`
}

// StaffOverviewResponse: board + box office + derived awaiting count, one call for the floor app.
type StaffOverviewResponse struct {
	Date              string                  `json:"date" example:"2026-09-14"`
	Showtimes         []StaffShowtimeResponse `json:"showtimes"`
	CounterSalesCount int64                   `json:"counter_sales_count"`
	CounterSalesTotal int64                   `json:"counter_sales_total"`
	AwaitingCheckin   int                     `json:"awaiting_checkin"`
}

// AdminStatsResponse: four headline counts. Excludes soft-deleted; bookings counts
// CONFIRMED only; locked users still counted (lock ≠ deletion, same as GET /admin/users).
type AdminStatsResponse struct {
	Movies    int64 `json:"movies"`
	Showtimes int64 `json:"showtimes"`
	Bookings  int64 `json:"bookings"`
	Users     int64 `json:"users"`
}

// BreakdownResponse: revenue analytics over PAID money in [from, to] - top
// movies/halls, payment-method split and the daily line. Same money rule as
// closeDay (confirmed bookings by payment time), aggregated live instead of
// waiting for closed rows, so charts work for any range including today.
type BreakdownMovie struct {
	MovieID  string `json:"movie_id"`
	Title    string `json:"title"`
	Revenue  int64  `json:"revenue"`
	Tickets  int    `json:"tickets"`
}

type BreakdownHall struct {
	HallID  string `json:"hall_id"`
	Name    string `json:"name"`
	Revenue int64  `json:"revenue"`
	Tickets int    `json:"tickets"`
}

type BreakdownProvider struct {
	Provider string `json:"provider"`
	Revenue  int64  `json:"revenue"`
	Count    int    `json:"count"`
}

type BreakdownDay struct {
	Date    string `json:"date" example:"2026-09-14"`
	Revenue int64  `json:"revenue"`
	Tickets int    `json:"tickets"`
}

type BreakdownResponse struct {
	From         string              `json:"from" example:"2026-09-08"`
	To           string              `json:"to" example:"2026-09-14"`
	TotalRevenue int64               `json:"total_revenue"`
	TicketsSold  int                 `json:"tickets_sold"`
	Days         []BreakdownDay      `json:"days"`
	Movies       []BreakdownMovie    `json:"movies"`
	Halls        []BreakdownHall     `json:"halls"`
	Providers    []BreakdownProvider `json:"providers"`
}
