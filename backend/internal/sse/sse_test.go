package sse

import (
	"testing"
	"time"
)

func event(show, seat, status string) SeatEvent {
	return SeatEvent{ShowtimeID: show, Seats: []SeatUpdate{{ID: seat, Status: status}}}
}

func subscribe(t *testing.T, hub *Hub, show, user string) *Subscription {
	t.Helper()
	sub, err := hub.Subscribe(show, user)
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

// T19 / R-S10: viewers of show A never receive show B's events.
func TestHub_IsolatesShowtimes(t *testing.T) {
	hub := NewHub()
	a := subscribe(t, hub, "show-a", "u1")
	defer a.Close()
	b := subscribe(t, hub, "show-b", "u2")
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

func TestHub_DropsSlowClient(t *testing.T) {
	hub := NewHub()
	slow := subscribe(t, hub, "show", "u1")
	defer slow.Close()
	fast := subscribe(t, hub, "show", "u2")
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
	sub := subscribe(t, hub, "show", "u1")
	sub.Close()
	sub.Close()
	if hub.Viewers("show") != 0 {
		t.Fatal("closed subscription still counted")
	}

	live := subscribe(t, hub, "show", "u1")
	hub.Close()
	select {
	case <-live.Done:
	case <-time.After(time.Second):
		t.Fatal("hub.Close did not end the stream")
	}
	late := subscribe(t, hub, "show", "u1")
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

	for i := 0; i < 2; i++ {
		if hall, user, ok := store.Validate(token, "show-a"); !ok || hall != "hall-1" || user != "user-1" {
			t.Fatalf("attempt %d: ok=%v hall=%q user=%q", i, ok, hall, user)
		}
	}
	if _, _, ok := store.Validate(token, "show-b"); ok {
		t.Fatal("token opened another showtime")
	}
	if _, _, ok := store.Validate("forged", "show-a"); ok {
		t.Fatal("unknown token accepted")
	}

	now = now.Add(29 * time.Second)
	if _, _, ok := store.Validate(token, "show-a"); !ok {
		t.Fatal("token rejected before expiry")
	}
	now = now.Add(time.Second)
	if _, _, ok := store.Validate(token, "show-a"); ok {
		t.Fatal("expired token accepted")
	}
}

func TestHub_StreamLimits(t *testing.T) {
	hub := NewHub()
	hub.MaxStreamsPerUser, hub.MaxStreams = 2, 3
	a1 := subscribe(t, hub, "show", "a")
	subscribe(t, hub, "show", "a")
	if _, err := hub.Subscribe("show", "a"); err != ErrTooManyStreams {
		t.Fatalf("third stream of one user: err = %v", err)
	}
	subscribe(t, hub, "other", "b")
	if _, err := hub.Subscribe("show", "c"); err != ErrTooManyStreams {
		t.Fatalf("stream over the server limit: err = %v", err)
	}
	a1.Close()
	a1.Close()
	if _, err := hub.Subscribe("show", "a"); err != nil {
		t.Fatalf("stream after closing one: %v", err)
	}
}

func TestTokenStore_PrunesLazily(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	store := NewTokenStore(30*time.Second, func() time.Time { return now })
	store.Issue("u", "s", "h")
	store.Issue("u", "s", "h")
	now = now.Add(31 * time.Second)
	store.Issue("u", "s", "h")
	if n := store.size(); n != 1 {
		t.Fatalf("tokens after sweep = %d, want 1", n)
	}
	now = now.Add(31 * time.Second)
	store.Issue("u", "s", "h")
	if n := store.size(); n != 1 {
		t.Fatalf("tokens after second sweep = %d, want 1", n)
	}
}
