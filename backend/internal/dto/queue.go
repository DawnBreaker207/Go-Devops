package dto

// SetQueueEnabledRequest is the admin manual activation switch (Phần 2.3/3).
type SetQueueEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type QueueJoinRequest struct {
	ShowtimeID string `json:"showtime_id" binding:"required"`
}

type QueueJoinResponse struct {
	Position int64 `json:"position"`
}

// QueueStatusResponse: Admitted=true means Hold will now accept this user
// for this showtime for a limited window (same TTL as a normal hold);
// Position is 0-based (0 = next in line) and only meaningful while waiting.
type QueueStatusResponse struct {
	Admitted bool  `json:"admitted"`
	Position int64 `json:"position,omitempty"`
}
