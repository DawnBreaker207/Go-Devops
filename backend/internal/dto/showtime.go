package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type ShowtimeRequest struct {
	MovieID string    `json:"movie_id" binding:"required,uuid" example:"10000000-0000-0000-0000-000000000001"`
	HallID  string    `json:"hall_id" binding:"required,uuid"`
	StartAt time.Time `json:"start_at" binding:"required" example:"2026-09-15T19:00:00+07:00"`
	// Status is optional; omitted keeps the current value on update.
	Status string `json:"status" binding:"omitempty,oneof=open closed"`
}

type ShowtimeResponse struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movie_id"`
	MovieTitle string    `json:"movie_title"`
	AgeRating  string    `json:"age_rating"`
	HallID     string    `json:"hall_id"`
	HallName   string    `json:"hall_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ShowtimeAdminListQuery: operator list hides nothing (closed/past/draft visible for fixing).
// Search matches movie title or hall name.
type ShowtimeAdminListQuery struct {
	PageQuery
	MovieID string `form:"movie_id" binding:"omitempty,uuid"`
	HallID  string `form:"hall_id" binding:"omitempty,uuid"`
	Status  string `form:"status" binding:"omitempty,oneof=open closed"`
	// Date narrows to a single calendar day in the server timezone and wins over From/To.
	Date  string `form:"date" binding:"omitempty,datetime=2006-01-02"`
	From  string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To    string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	Sort  string `form:"sort" binding:"omitempty,oneof=start_at created_at"`
	Order string `form:"order" binding:"omitempty,oneof=asc desc"`
}

type ShowtimeListItem struct {
	ID         string    `json:"id"`
	MovieID    string    `json:"movie_id"`
	MovieTitle string    `json:"movie_title"`
	AgeRating  string    `json:"age_rating"`
	HallID     string    `json:"hall_id"`
	HallName   string    `json:"hall_name"`
	StartAt    time.Time `json:"start_at"`
	EndAt      time.Time `json:"end_at"`
	Status     string    `json:"status"`
	FromPrice  int64     `json:"from_price,omitempty"`
}

func NewShowtimeListItem(s models.Showtime, hallName string, fromPrice int64) ShowtimeListItem {
	return ShowtimeListItem{
		ID:        s.ID,
		MovieID:   s.MovieID,
		HallID:    s.HallID,
		HallName:  hallName,
		StartAt:   s.StartAt,
		EndAt:     s.EndAt,
		Status:    s.Status,
		FromPrice: fromPrice,
	}
}

// ShowtimeCancelResponse: bookings refunded through the normal refund pipeline.
type ShowtimeCancelResponse struct {
	ShowtimeID       string `json:"showtime_id"`
	Status           string `json:"status"`
	BookingsAffected int    `json:"bookings_affected"`
}

type SeatMapSeat struct {
	ID             string `json:"id"`
	ShowtimeSeatID string `json:"showtime_seat_id,omitempty"`
	Label          string `json:"label"`
	RowLabel       string `json:"row_label"`
	Col            int    `json:"col_number"`
	SeatType       string `json:"seat_type"`
	IsGap          bool   `json:"is_gap"`
	ColSpan        int    `json:"col_span"`
	Status         string `json:"status"`
	Price          int64  `json:"price"`
}

type SeatMapResponse struct {
	ShowtimeID     string           `json:"showtime_id"`
	MovieID        string           `json:"movie_id"`
	MovieTitle     string           `json:"movie_title"`
	AgeRating      string           `json:"age_rating"`
	HallID         string           `json:"hall_id"`
	HallName       string           `json:"hall_name"`
	StartAt        time.Time        `json:"start_at"`
	EndAt          time.Time        `json:"end_at"`
	Status         string           `json:"status"`
	ScreenPosition string           `json:"screen_position"`
	AisleAfterCols []int            `json:"aisle_after_cols"`
	Prices         map[string]int64 `json:"prices"`
	Seats          []SeatMapSeat    `json:"seats"`
}
