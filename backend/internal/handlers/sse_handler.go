package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// ShowtimeLookup resolves an open showtime for token issuance.
type ShowtimeLookup interface {
	OpenShowtime(ctx context.Context, id string) (*dto.ShowtimeResponse, error)
}

// SSEHandler serves the realtime seat-map stream (F11).
type SSEHandler struct {
	hub       *sse.Hub
	tokens    *sse.TokenStore
	showtimes ShowtimeLookup
	debounce  time.Duration
	keepalive time.Duration
	// writeTimeout bounds every single write: a client that stops reading
	// frees its connection instead of holding a goroutine forever (M13).
	writeTimeout time.Duration
	// maxAge ends a stream so the client comes back through /events/token,
	// which rechecks the account and the showtime.
	maxAge time.Duration
}

func NewSSEHandler(hub *sse.Hub, tokens *sse.TokenStore, showtimes ShowtimeLookup) *SSEHandler {
	return &SSEHandler{
		hub:          hub,
		tokens:       tokens,
		showtimes:    showtimes,
		debounce:     100 * time.Millisecond,
		keepalive:    15 * time.Second,
		writeTimeout: 10 * time.Second,
		maxAge:       30 * time.Minute,
	}
}

// seatsPayload is the data of one "seats" event.
type seatsPayload struct {
	ShowtimeID string           `json:"showtime_id"`
	HallID     string           `json:"hall_id"`
	Seats      []sse.SeatUpdate `json:"seats"`
}

// IssueToken godoc
//
//	@Summary		Create a short-lived realtime token for one showtime
//	@Description	The token (not the JWT) goes into the EventSource URL. Valid 30s, reusable for reconnects, bound to the showtime.
//	@Tags			events
//	@Produce		json
//	@Security		BearerAuth
//	@Param			show_id	query	string	true	"Showtime ID"
//	@Success		200		{object}	response.Body{data=map[string]any}
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/events/token [get]
func (h *SSEHandler) IssueToken(c *gin.Context) {
	showID := c.Query("show_id")
	if showID == "" {
		response.Error(c, apperrors.Validation("show_id is required"))
		return
	}
	showtime, err := h.showtimes.OpenShowtime(c.Request.Context(), showID)
	if err != nil {
		response.Error(c, err)
		return
	}
	token, ttl := h.tokens.Issue(middleware.CurrentUserID(c), showtime.ID, showtime.HallID)
	response.OK(c, gin.H{
		"token":      token,
		"expires_in": int(ttl.Seconds()),
		"stream_url": fmt.Sprintf("/api/v1/events/shows/%s?token=%s", showtime.ID, token),
	})
}

// Stream godoc
//
//	@Summary		SSE stream of seat changes for a showtime
//	@Description	No JWT: the realtime token is the credential. Events: connected, seats (debounced ~100ms); ": ping" every 15s.
//	@Tags			events
//	@Produce		text/event-stream
//	@Param			id		path	string	true	"Showtime ID"
//	@Param			token	query	string	true	"Realtime token from /events/token"
//	@Failure		401		{object}	response.Body
//	@Router			/events/shows/{id} [get]
func (h *SSEHandler) Stream(c *gin.Context) {
	showtimeID := c.Param("id")
	hallID, userID, ok := h.tokens.Validate(c.Query("token"), showtimeID)
	if !ok {
		// R-S3: the page requests a fresh token and reconnects.
		response.Error(c, apperrors.Unauthorized("invalid or expired realtime token"))
		return
	}

	sub, err := h.hub.Subscribe(showtimeID, userID)
	if err != nil {
		response.Error(c, apperrors.TooManyRequests("too many realtime streams open; close another tab"))
		return
	}
	defer sub.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // R-S1: nginx must not buffer
	c.Status(http.StatusOK)

	// The server-wide WriteTimeout would cut this long-lived response; every
	// write gets its own short deadline instead, so a stalled client is let go.
	rc := http.NewResponseController(c.Writer)
	w := c.Writer
	write := func(chunk string) bool {
		_ = rc.SetWriteDeadline(time.Now().Add(h.writeTimeout))
		if _, err := w.WriteString(chunk); err != nil {
			return false
		}
		return rc.Flush() == nil
	}
	event := func(name string, data any) bool {
		body, err := json.Marshal(data)
		if err != nil {
			return false
		}
		return write(fmt.Sprintf("event: %s\ndata: %s\n\n", name, body))
	}

	// R-S8: clients reconnecting together spread out over 3 seconds.
	if !write("retry: 3000\n\n") || !event("connected", gin.H{"showtime_id": showtimeID, "hall_id": hallID}) {
		return
	}

	keepalive := time.NewTicker(h.keepalive)
	defer keepalive.Stop()
	expire := time.NewTimer(h.maxAge)
	defer expire.Stop()

	var (
		pending []sse.SeatUpdate
		index   = map[string]int{}
		timer   *time.Timer
		flush   <-chan time.Time
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-sub.Done:
			return
		case <-expire.C:
			return // retry: 3000 brings the client back through a fresh token
		case <-keepalive.C:
			if !write(": ping\n\n") {
				return
			}
		case ev := <-sub.Events:
			// Debounce: coalesce changes for a short window, last status wins.
			for _, u := range ev.Seats {
				if i, seen := index[u.ID]; seen {
					pending[i] = u
					continue
				}
				index[u.ID] = len(pending)
				pending = append(pending, u)
			}
			if timer == nil {
				timer = time.NewTimer(h.debounce)
				flush = timer.C
			}
		case <-flush:
			if !event("seats", seatsPayload{ShowtimeID: showtimeID, HallID: hallID, Seats: pending}) {
				return
			}
			pending, index, timer, flush = nil, map[string]int{}, nil, nil
		}
	}
}
