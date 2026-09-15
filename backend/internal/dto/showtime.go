package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// ShowtimeRequest schedules a movie in a hall.
type ShowtimeRequest struct {
	MovieID string    `json:"movie_id" binding:"required,uuid" example:"10000000-0000-0000-0000-000000000001"`
	HallID  string    `json:"hall_id" binding:"required,uuid"`
	StartAt time.Time `json:"start_at" binding:"required" example:"2026-09-15T19:00:00+07:00"`
	// Status is optional; omitted keeps the current value on update.
	Status string `json:"status" binding:"omitempty,oneof=open closed"`
}

// ShowtimeResponse is a scheduled showtime returned to the client.
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

// ShowtimeListItem is one showtime in the movie picker.
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

// NewShowtimeListItem maps a row with hall name and price to a DTO.
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

// SeatMapSeat is one seat inside the showtime grid.
type SeatMapSeat struct {
	ID             string `json:"id"`
	ShowtimeSeatID string `json:"showtime_seat_id,omitempty"`
	Label          string `json:"label"`
	RowLabel       string `json:"row_label"`
	Col            int    `json:"col_number"`
	SeatType       string `json:"seat_type"`
	IsGap          bool   `json:"is_gap"`
	Status         string `json:"status"`
	Price          int64  `json:"price"`
}

// SeatMapResponse is the seat grid for a showtime.
type SeatMapResponse struct {
	ShowtimeID string         `json:"showtime_id"`
	MovieID    string         `json:"movie_id"`
	MovieTitle string         `json:"movie_title"`
	AgeRating  string         `json:"age_rating"`
	HallID     string         `json:"hall_id"`
	HallName   string         `json:"hall_name"`
	StartAt    time.Time      `json:"start_at"`
	EndAt      time.Time      `json:"end_at"`
	Status     string         `json:"status"`
	Prices     map[string]int64 `json:"prices"`
	Seats      []SeatMapSeat  `json:"seats"`
}