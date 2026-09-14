package dto

import (
	"strconv"
	"strings"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// HallRequest creates a hall; the seat grid is generated from these params.
type HallRequest struct {
	Name        string              `json:"name" binding:"required,min=1,max=255" example:"Phong 2"`
	Rows        int                 `json:"rows" binding:"required,min=1,max=50" example:"8"`
	SeatsPerRow int                 `json:"seats_per_row" binding:"required,min=1,max=50" example:"12"`
SeatTypes map[string][]string `json:"seat_types" binding:"required" example:"vip:7,8"`
	Gaps        []string            `json:"gaps" binding:"omitempty,max=200" example:"[\"D5\",\"D6\"]"`
}

// PriceRequest sets the price for every seat type of a hall.
type PriceRequest struct {
	Prices map[string]int64 `json:"prices" binding:"required" example:"standard:80000,vip:120000"`
}

// SeatUpdateRequest changes one seat's type or gap flag.
type SeatUpdateRequest struct {
	SeatType string `json:"seat_type" binding:"omitempty,oneof=standard vip couple recliner"`
	IsGap    bool   `json:"is_gap"`
}

// HallResponse is a hall returned to the client.
type HallResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Rows        int                 `json:"rows"`
	SeatsPerRow int                 `json:"seats_per_row"`
	SeatTypes   map[string][]string `json:"seat_types"`
	Gaps        []string            `json:"gaps"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func NewHallResponse(hall *models.Hall) HallResponse {
	return HallResponse{
		ID:          hall.ID,
		Name:        hall.Name,
		Rows:        hall.Rows,
		SeatsPerRow: hall.SeatsPerRow,
		SeatTypes:   hall.SeatTypes,
		Gaps:        hall.Gaps,
		CreatedAt:   hall.CreatedAt,
		UpdatedAt:   hall.UpdatedAt,
	}
}

func NewHallResponses(halls []models.Hall) []HallResponse {
	result := make([]HallResponse, 0, len(halls))
	for i := range halls {
		result = append(result, NewHallResponse(&halls[i]))
	}
	return result
}

// SeatResponse is a seat returned to the client.
type SeatResponse struct {
	ID       string `json:"id"`
	HallID   string `json:"hall_id"`
	Label    string `json:"label"`
	RowLabel string `json:"row_label"`
	Col      int    `json:"col_number"`
	SeatType string `json:"seat_type"`
	IsGap    bool   `json:"is_gap"`
}

// SeatLabel builds a label such as "D5" from a row label and column number.
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
	}
}

func NewSeatResponses(seats []models.Seat) []SeatResponse {
	result := make([]SeatResponse, 0, len(seats))
	for i := range seats {
		result = append(result, NewSeatResponse(&seats[i]))
	}
	return result
}

// HallPriceResponse is a single seat-type price for a hall.
type HallPriceResponse struct {
	SeatType string `json:"seat_type"`
	Price    int64  `json:"price"`
}

// RowNumber parses an Excel-like row label ("A" -> 1) into its index.
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

// RowLabel converts a 1-based row index into an Excel-like label (1 -> "A").
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