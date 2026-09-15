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
