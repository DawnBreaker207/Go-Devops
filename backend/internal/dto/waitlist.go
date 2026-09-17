package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type WaitlistJoinRequest struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
}

type WaitlistEntryResponse struct {
	ID                 string     `json:"id"`
	ShowtimeID         string     `json:"showtime_id"`
	Status             string     `json:"status"`
	FulfilledBookingID string     `json:"fulfilled_booking_id,omitempty"`
	NotifiedAt         *time.Time `json:"notified_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

func NewWaitlistEntryResponse(e *models.WaitlistEntry) WaitlistEntryResponse {
	r := WaitlistEntryResponse{
		ID: e.ID, ShowtimeID: e.ShowtimeID, Status: e.Status,
		NotifiedAt: e.NotifiedAt, ExpiresAt: e.ExpiresAt, CreatedAt: e.CreatedAt,
	}
	if e.FulfilledBookingID != nil {
		r.FulfilledBookingID = *e.FulfilledBookingID
	}
	return r
}

func NewWaitlistEntryResponses(rows []models.WaitlistEntry) []WaitlistEntryResponse {
	out := make([]WaitlistEntryResponse, 0, len(rows))
	for i := range rows {
		out = append(out, NewWaitlistEntryResponse(&rows[i]))
	}
	return out
}
