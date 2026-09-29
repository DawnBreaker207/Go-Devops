package dto

import (
	"strconv"
	"strings"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// With Template set, explicit fields override its defaults.
type HallRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255" example:"Phong 2"`
	Template    string `json:"template" binding:"omitempty,oneof=small medium large" example:"medium"`
	Rows        int    `json:"rows" binding:"omitempty,min=1,max=50" example:"8"`
	SeatsPerRow int    `json:"seats_per_row" binding:"omitempty,min=1,max=50" example:"12"`
	SeatTypes map[string][]string `json:"seat_types"`
	Gaps      []string            `json:"gaps" binding:"omitempty,max=200" example:"[\"D5\",\"D6\"]"`
	// Anchors of 2-column seats, e.g. ["D3"] spans D3-D4; the neighbor keeps no seat.
	Spans          []string `json:"spans" binding:"omitempty,max=100" example:"[\"D3\"]"`
	ScreenPosition string   `json:"screen_position" binding:"omitempty,oneof=front back" example:"front"`
	// Display only.
	AisleAfterCols []int `json:"aisle_after_cols" binding:"omitempty,max=49" example:"4"`
	// Nil defaults to true.
	Active *bool `json:"active"`
}

type UpdateHallRequest struct {
	Name           *string `json:"name" binding:"omitempty,min=1,max=255"`
	ScreenPosition *string `json:"screen_position" binding:"omitempty,oneof=front back"`
	AisleAfterCols []int   `json:"aisle_after_cols" binding:"omitempty,max=49"`
	Active         *bool   `json:"active"`
}

// Copies the full seat grid, incl. manual edits.
type CloneHallRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255" example:"Phong 3"`
}

// Exactly one field.
type SeatSelector struct {
	Labels []string `json:"labels" binding:"omitempty,max=500" example:"[\"A1\",\"A2\"]"`
	Rows   []string `json:"rows" binding:"omitempty,max=50" example:"[\"A\",\"B\"]"`
	Cols   []int    `json:"cols" binding:"omitempty,max=50" example:"1"`
	Range string `json:"range" binding:"omitempty,max=16" example:"A1:C4"`
}

type SeatChange struct {
	Selector SeatSelector `json:"selector" binding:"required"`
	SeatType string       `json:"seat_type" binding:"omitempty,oneof=standard vip couple recliner"`
	IsGap    *bool        `json:"is_gap"`
}

// One transaction; one failure rolls all back.
type BulkSeatUpdateRequest struct {
	Changes []SeatChange `json:"changes" binding:"required,min=1,max=50,dive"`
}

type HallTemplateResponse struct {
	Name        string         `json:"name"`
	Rows        int            `json:"rows"`
	SeatsPerRow int            `json:"seats_per_row"`
	SeatCount   int            `json:"seat_count"`
	ByType      map[string]int `json:"seat_count_by_type"`
}

type SeatUpdateRequest struct {
	SeatType string `json:"seat_type" binding:"omitempty,oneof=standard vip couple recliner"`
	IsGap    *bool  `json:"is_gap"`
}

// Two adjacent standards become one couple; right seat deleted.
type MergeSeatsRequest struct {
	LeftLabel  string `json:"left_label" binding:"required,max=8" example:"D3"`
	RightLabel string `json:"right_label" binding:"required,max=8" example:"D4"`
}

// New seat at next column.
type SplitSeatRequest struct {
	Label string `json:"label" binding:"required,max=8" example:"D3"`
}

// One seat appended at the end of a single row (the row-end "+" in the
// editor). Couple seats come only from merging, never from direct creation.
type AddSeatRequest struct {
	RowLabel string `json:"row_label" binding:"required,max=8" example:"B"`
	SeatType string `json:"seat_type" binding:"omitempty,oneof=standard vip recliner" example:"standard"`
}

type HallResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Rows           int       `json:"rows"`
	SeatsPerRow    int       `json:"seats_per_row"`
	ScreenPosition string    `json:"screen_position"`
	AisleAfterCols []int     `json:"aisle_after_cols"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func NewHallResponse(hall *models.Hall) HallResponse {
	return HallResponse{
		ID:             hall.ID,
		Name:           hall.Name,
		Rows:           hall.Rows,
		SeatsPerRow:    hall.SeatsPerRow,
		ScreenPosition: hall.ScreenPosition,
		AisleAfterCols: hall.AisleAfterCols,
		Active:         hall.Active,
		CreatedAt:      hall.CreatedAt,
		UpdatedAt:      hall.UpdatedAt,
	}
}

func NewHallResponses(halls []models.Hall) []HallResponse {
	result := make([]HallResponse, 0, len(halls))
	for i := range halls {
		result = append(result, NewHallResponse(&halls[i]))
	}
	return result
}

type SeatResponse struct {
	ID       string `json:"id"`
	HallID   string `json:"hall_id"`
	Label    string `json:"label"`
	RowLabel string `json:"row_label"`
	Col      int    `json:"col_number"`
	SeatType string `json:"seat_type"`
	IsGap    bool   `json:"is_gap"`
	ColSpan  int    `json:"col_span"`
	// True when the seat ever appeared in a booking: merge/split refuse it.
	HasBookingHistory bool `json:"has_booking_history"`
}

func SeatLabel(rowLabel string, col int) string {
	return rowLabel + strconv.Itoa(col)
}

func NewSeatResponse(seat *models.Seat) SeatResponse {
	return SeatResponse{
		ID:       seat.ID,
		HallID:   seat.HallID,
		Label:    SeatLabel(seat.RowLabel, seat.ColNumber),
		RowLabel: seat.RowLabel,
		Col:      seat.ColNumber,
		SeatType: seat.SeatType,
		IsGap:    seat.IsGap,
		ColSpan:  seat.ColSpan,
		HasBookingHistory: seat.HasBookingHistory,
	}
}

func NewSeatResponses(seats []models.Seat) []SeatResponse {
	result := make([]SeatResponse, 0, len(seats))
	for i := range seats {
		result = append(result, NewSeatResponse(&seats[i]))
	}
	return result
}

// 0 for an invalid label.
func RowNumber(label string) int {
	n := 0
	for _, ch := range strings.ToUpper(label) {
		if ch < 'A' || ch > 'Z' {
			return 0
		}
		n = n*26 + int(ch-'A'+1)
	}
	return n
}

func RowLabel(n int) string {
	var b strings.Builder
	for n > 0 {
		n--
		b.WriteByte(byte('A' + n%26))
		n /= 26
	}
	s := []byte(b.String())
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return string(s)
}
