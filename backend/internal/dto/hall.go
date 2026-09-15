package dto

import (
	"strconv"
	"strings"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// The seat grid is generated from these params. With Template set, Rows,
// SeatsPerRow, SeatTypes, Gaps, Spans, ScreenPosition and AisleAfterCols
// default to the template's own layout; any of them given explicitly still
// override it.
type HallRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255" example:"Phong 2"`
	Template    string `json:"template" binding:"omitempty,oneof=small medium large" example:"medium"`
	Rows        int    `json:"rows" binding:"omitempty,min=1,max=50" example:"8"`
	SeatsPerRow int    `json:"seats_per_row" binding:"omitempty,min=1,max=50" example:"12"`
	// Optional: rows not listed are standard.
	SeatTypes map[string][]string `json:"seat_types"`
	Gaps      []string            `json:"gaps" binding:"omitempty,max=200" example:"[\"D5\",\"D6\"]"`
	// Anchors of 2-column seats, e.g. ["D3"] makes one seat spanning D3-D4; the
	// neighbor column keeps no seat of its own.
	Spans          []string `json:"spans" binding:"omitempty,max=100" example:"[\"D3\"]"`
	ScreenPosition string   `json:"screen_position" binding:"omitempty,oneof=front back" example:"front"`
	// Column numbers after which there is a vertical aisle, display only.
	AisleAfterCols []int `json:"aisle_after_cols" binding:"omitempty,max=49" example:"[4,8]"`
	// Must hold a positive price for each of the 4 seat types.
	Prices map[string]int64 `json:"prices" binding:"required" example:"standard:70000,vip:100000,couple:160000,recliner:130000"`
}

// UpdateHallRequest changes a hall's name, screen/aisle display and whether it
// still takes new showtimes. Omitted fields keep their current value.
type UpdateHallRequest struct {
	Name           *string `json:"name" binding:"omitempty,min=1,max=255"`
	ScreenPosition *string `json:"screen_position" binding:"omitempty,oneof=front back"`
	AisleAfterCols []int   `json:"aisle_after_cols" binding:"omitempty,max=49"`
	Active         *bool   `json:"active"`
}

// CloneHallRequest copies a hall's full seat grid (including any manual seat
// edits) under a new name.
type CloneHallRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=255" example:"Phong 3"`
	CopyPrices bool   `json:"copy_prices"`
}

// SeatSelector picks which seats a bulk change applies to; exactly one field.
type SeatSelector struct {
	Labels []string `json:"labels" binding:"omitempty,max=500" example:"[\"A1\",\"A2\"]"`
	Rows   []string `json:"rows" binding:"omitempty,max=50" example:"[\"A\",\"B\"]"`
	Cols   []int    `json:"cols" binding:"omitempty,max=50" example:"[1,2]"`
	// Inclusive rectangle, e.g. "A1:C4".
	Range string `json:"range" binding:"omitempty,max=16" example:"A1:C4"`
}

type SeatChange struct {
	Selector SeatSelector `json:"selector" binding:"required"`
	SeatType string       `json:"seat_type" binding:"omitempty,oneof=standard vip couple recliner"`
	IsGap    *bool        `json:"is_gap"`
}

// BulkSeatUpdateRequest applies every change in one transaction; one failing
// change rolls the whole batch back.
type BulkSeatUpdateRequest struct {
	Changes []SeatChange `json:"changes" binding:"required,min=1,max=50,dive"`
}

// HallTemplateResponse previews a built-in hall template before creating it.
type HallTemplateResponse struct {
	Name        string         `json:"name"`
	Rows        int            `json:"rows"`
	SeatsPerRow int            `json:"seats_per_row"`
	SeatCount   int            `json:"seat_count"`
	ByType      map[string]int `json:"seat_count_by_type"`
}

type PriceRequest struct {
	Prices map[string]int64 `json:"prices" binding:"required" example:"standard:80000,vip:120000"`
}

type SeatUpdateRequest struct {
	SeatType string `json:"seat_type" binding:"omitempty,oneof=standard vip couple recliner"`
	// Omitted keeps the current value.
	IsGap *bool `json:"is_gap"`
}

type HallResponse struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Rows           int                 `json:"rows"`
	SeatsPerRow    int                 `json:"seats_per_row"`
	SeatTypes      map[string][]string `json:"seat_types"`
	Gaps           []string            `json:"gaps"`
	ScreenPosition string              `json:"screen_position"`
	AisleAfterCols []int               `json:"aisle_after_cols"`
	Active         bool                `json:"active"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

func NewHallResponse(hall *models.Hall) HallResponse {
	return HallResponse{
		ID:             hall.ID,
		Name:           hall.Name,
		Rows:           hall.Rows,
		SeatsPerRow:    hall.SeatsPerRow,
		SeatTypes:      hall.SeatTypes,
		Gaps:           hall.Gaps,
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
	}
}

func NewSeatResponses(seats []models.Seat) []SeatResponse {
	result := make([]SeatResponse, 0, len(seats))
	for i := range seats {
		result = append(result, NewSeatResponse(&seats[i]))
	}
	return result
}

type HallPriceResponse struct {
	SeatType string `json:"seat_type"`
	Price    int64  `json:"price"`
}

// Excel-like row label to index: "A" -> 1, "AA" -> 27; 0 for an invalid label.
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

// 1-based row index to Excel-like label: 1 -> "A", 27 -> "AA".
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
