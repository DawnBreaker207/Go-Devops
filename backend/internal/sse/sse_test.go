package sse

import (
	"testing"
	"time"
)

func event(show, seat, status string) SeatEvent {
	return SeatEvent{ShowtimeID: show, Seats: []SeatUpdate{{ID: seat, Status: status}}}
}

// T19 / R-S10: viewers of show A never receive show B's events.
func TestHub_IsolatesShowtimes(t *testing.T) {
	hub := NewHub()
	a := hub.Subscribe("show-a")
	defer a.Close()
	b := hub.Subscribe("show-b")
	defer b.Close()

	hub.Broadcast("show-b", event("show-b", "s1", "held"))

	select {
	case ev := <-a.Events:
		t.Fatalf("show-a viewer got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
	select {
	case ev := <-b.Events:
		if ev.ShowtimeID != "show-b" {
			t.Fatalf("got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("show-b viewer got nothing")
	}
}

// R-S4: a client that stops reading is dropped, others keep receiving.
func TestHub_DropsSlowClient(t *testing.T) {
	hub := NewHub()
	slow := hub.Subscribe("show")
	defer slow.Close()
	fast := hub.Subscribe("show")
	defer fast.Close()

	for i := 0; i < clientBuffer+1; i++ {
		hub.Broadcast("show", event("show", "s", "held"))
		<-fast.Events
	}
	select {
	case <-slow.Done:
	default:
		t.Fatal("slow client was not dropped")
	}
	select {
	case <-fast.Done:
		t.Fatal("fast client was dropped")
	default:
	}
}

func TestHub_CloseEndsSubscriptions(t *testing.T) {
	hub := NewHub()
	sub := hub.Subscribe("show")
	sub.Close()
	sub.Close() // idempotent
	if hub.Viewers("show") != 0 {
		t.Fatal("closed subscription still counted")
	}

	live := hub.Subscribe("show")
	hub.Close()
	select {
	case <-live.Done:
	case <-time.After(time.Second):
		t.Fatal("hub.Close did not end the stream")
	}
	late := hub.Subscribe("show")
	select {
	case <-late.Done:
	default:
		t.Fatal("subscription after Close should end immediately")
	}
}

// T20 / R-S2: token bound to one showtime, reusable until it expires.
func TestTokenStore(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	store := NewTokenStore(30*time.Second, func() time.Time { return now })
	token, ttl := store.Issue("user-1", "show-a", "hall-1")
	if ttl != 30*time.Second {
		t.Fatalf("ttl = %v", ttl)
	}

	for i := 0; i < 2; i++ { // reconnect with the same URL
		if hall, ok := store.Validate(token, "show-a"); !ok || hall != "hall-1" {
			t.Fatalf("attempt %d: ok=%v hall=%q", i, ok, hall)
		}
	}
	if _, ok := store.Validate(token, "show-b"); ok {
		t.Fatal("token opened another showtime")
	}
	if _, ok := store.Validate("forged", "show-a"); ok {
		t.Fatal("unknown token accepted")
	}

	now = now.Add(29 * time.Second)
	if _, ok := store.Validate(token, "show-a"); !ok {
		t.Fatal("token rejected before expiry")
	}
	now = now.Add(time.Second)
	if _, ok := store.Validate(token, "show-a"); ok {
		t.Fatal("expired token accepted")
	}
}
