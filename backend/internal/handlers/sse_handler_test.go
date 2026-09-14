package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type fakeShowtimes struct{}

func (fakeShowtimes) OpenShowtime(_ context.Context, id string) (*dto.ShowtimeResponse, error) {
	if id == "show-a" || id == "show-b" {
		return &dto.ShowtimeResponse{ID: id, HallID: "hall-1"}, nil
	}
	return nil, apperrors.ErrShowtimeNotFound
}

type sseFrame struct {
	event string
	data  string
}

func startSSEServer(t *testing.T, writeTimeout time.Duration, tune ...func(*SSEHandler)) (*httptest.Server, *sse.Hub, *sse.TokenStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	hub := sse.NewHub()
	tokens := sse.NewTokenStore(time.Minute, nil)
	h := NewSSEHandler(hub, tokens, fakeShowtimes{})
	h.debounce = 60 * time.Millisecond
	h.keepalive = 150 * time.Millisecond
	for _, fn := range tune {
		fn(h)
	}

	engine := gin.New()
	engine.GET("/events/shows/:id", h.Stream)
	srv := httptest.NewUnstartedServer(engine)
	srv.Config.WriteTimeout = writeTimeout
	srv.Start()
	t.Cleanup(srv.Close)
	return srv, hub, tokens
}

// readFrames parses the stream into events; comment lines become ":" frames
// and the retry field a "retry" frame.
func readFrames(body *bufio.Reader, out chan<- sseFrame) {
	defer close(out)
	var cur sseFrame
	for {
		line, err := body.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if cur.event != "" || cur.data != "" {
				out <- cur
			}
			cur = sseFrame{}
		case strings.HasPrefix(line, ":"):
			out <- sseFrame{event: ":", data: strings.TrimSpace(line[1:])}
		case strings.HasPrefix(line, "retry: "):
			out <- sseFrame{event: "retry", data: line[len("retry: "):]}
		case strings.HasPrefix(line, "event: "):
			cur.event = line[len("event: "):]
		case strings.HasPrefix(line, "data: "):
			cur.data = line[len("data: "):]
		}
	}
}

func nextFrame(t *testing.T, frames <-chan sseFrame, want string, within time.Duration) sseFrame {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("stream closed while waiting for %q", want)
			}
			if f.event == want {
				return f
			}
		case <-deadline:
			t.Fatalf("no %q frame within %v", want, within)
		}
	}
}

// F11 end to end over HTTP: no JWT, token in URL, stream outlives the server
// WriteTimeout, keepalive pings, debounced batches, show isolation, and
// hub.Close ends the stream (graceful shutdown).
func TestSSEStream(t *testing.T) {
	srv, hub, tokens := startSSEServer(t, 300*time.Millisecond)
	token, _ := tokens.Issue("user-1", "show-a", "hall-1")

	resp, err := http.Get(srv.URL + "/events/shows/show-a?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d, content-type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatal("missing X-Accel-Buffering: no")
	}
	frames := make(chan sseFrame, 32)
	go readFrames(bufio.NewReader(resp.Body), frames)

	if f := nextFrame(t, frames, "retry", time.Second); f.data != "3000" {
		t.Fatalf("retry = %q", f.data)
	}
	nextFrame(t, frames, "connected", time.Second)

	// Past the 300ms WriteTimeout the stream must still deliver.
	time.Sleep(500 * time.Millisecond)
	nextFrame(t, frames, ":", time.Second) // keepalive ping

	hub.Broadcast("show-b", sse.SeatEvent{ShowtimeID: "show-b", Seats: []sse.SeatUpdate{{ID: "other", Status: "held"}}})
	hub.Broadcast("show-a", sse.SeatEvent{ShowtimeID: "show-a", Seats: []sse.SeatUpdate{{ID: "s1", Status: "held"}}})
	hub.Broadcast("show-a", sse.SeatEvent{ShowtimeID: "show-a", Seats: []sse.SeatUpdate{{ID: "s2", Status: "held"}}})
	hub.Broadcast("show-a", sse.SeatEvent{ShowtimeID: "show-a", Seats: []sse.SeatUpdate{{ID: "s1", Status: "sold"}}})

	f := nextFrame(t, frames, "seats", time.Second)
	var payload seatsPayload
	if err := json.Unmarshal([]byte(f.data), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.HallID != "hall-1" || payload.ShowtimeID != "show-a" {
		t.Fatalf("payload = %+v", payload)
	}
	want := []sse.SeatUpdate{{ID: "s1", Status: "sold"}, {ID: "s2", Status: "held"}}
	if len(payload.Seats) != len(want) || payload.Seats[0] != want[0] || payload.Seats[1] != want[1] {
		t.Fatalf("debounced seats = %+v, want %+v (show-b must not leak)", payload.Seats, want)
	}

	hub.Close()
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-frames:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("hub.Close did not end the stream")
		}
	}
}

// M13: a client that stops reading is let go after the write deadline instead
// of holding its goroutine and connection forever.
func TestSSEStream_StalledClientIsDropped(t *testing.T) {
	srv, hub, tokens := startSSEServer(t, 0, func(h *SSEHandler) {
		h.debounce = 5 * time.Millisecond
		h.writeTimeout = 200 * time.Millisecond
	})
	token, _ := tokens.Issue("user-1", "show-a", "hall-1")
	resp, err := http.Get(srv.URL + "/events/shows/show-a?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close() // never read: the client stalls

	// Distinct seat ids: the debounce keeps one update per seat, so the payload
	// stays large enough to fill the socket buffers of a client that stopped reading.
	seats := make([]sse.SeatUpdate, 4000)
	for i := range seats {
		seats[i] = sse.SeatUpdate{ID: fmt.Sprintf("%036d", i), Status: "held"}
	}
	deadline := time.Now().Add(20 * time.Second)
	for hub.Viewers("show-a") != 0 {
		if time.Now().After(deadline) {
			t.Fatal("stalled client still attached")
		}
		hub.Broadcast("show-a", sse.SeatEvent{ShowtimeID: "show-a", Seats: seats})
		time.Sleep(10 * time.Millisecond)
	}
}

// M13: a stream ends after its max age; the client reconnects with a new token.
func TestSSEStream_MaxAgeEndsStream(t *testing.T) {
	srv, _, tokens := startSSEServer(t, 0, func(h *SSEHandler) { h.maxAge = 300 * time.Millisecond })
	token, _ := tokens.Issue("user-1", "show-a", "hall-1")
	resp, err := http.Get(srv.URL + "/events/shows/show-a?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 1024)
		for {
			if _, err := resp.Body.Read(buf); err != nil {
				close(done)
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream outlived its max age")
	}
}

// M13: over the per-user stream limit the stream is refused with 429.
func TestSSEStream_PerUserLimit(t *testing.T) {
	srv, hub, tokens := startSSEServer(t, 0)
	hub.MaxStreamsPerUser = 1
	token, _ := tokens.Issue("user-1", "show-a", "hall-1")
	first, err := http.Get(srv.URL + "/events/shows/show-a?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Body.Close()
	if _, err := bufio.NewReader(first.Body).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	second, err := http.Get(srv.URL + "/events/shows/show-a?token=" + token)
	if err != nil {
		t.Fatal(err)
	}
	second.Body.Close()
	if second.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second stream: HTTP %d, want 429", second.StatusCode)
	}
}

// R-S2 / R-S3: a token for another show, a forged or missing token is 401.
func TestSSEStream_RejectsBadTokens(t *testing.T) {
	srv, _, tokens := startSSEServer(t, 0)
	token, _ := tokens.Issue("user-1", "show-a", "hall-1")

	for name, url := range map[string]string{
		"other show":    "/events/shows/show-b?token=" + token,
		"forged token":  "/events/shows/show-a?token=forged",
		"missing token": "/events/shows/show-a",
	} {
		resp, err := http.Get(srv.URL + url)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", name, resp.StatusCode)
		}
	}
}
