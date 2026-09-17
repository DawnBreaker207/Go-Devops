package service_test

import (
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

// TestWaitlist_JoinRefusedWhileSeatsAvailable: a fresh showtime always has open seats.
func TestWaitlist_JoinRefusedWhileSeatsAvailable(t *testing.T) {
	e := newEnv(t)
	svc := service.NewWaitlistService(e.db, repository.NewWaitlistRepository(e.db))
	if _, err := svc.Join(e.ctx, e.users[0], e.showID); err == nil {
		t.Fatal("want an error: seats are still available")
	}
}

// TestWaitlist_ReleasedSeatFulfillsOldestEntry: once every seat is taken,
// joining succeeds; when a held seat expires and the sweep releases it, the
// oldest waiting entry gets a PENDING booking for it (held on their behalf).
func TestWaitlist_ReleasedSeatFulfillsOldestEntry(t *testing.T) {
	e := newEnv(t)
	waitlistRepo := repository.NewWaitlistRepository(e.db)
	svc := service.NewWaitlistService(e.db, waitlistRepo)

	// Fill the whole hall (9 sellable seats) across a few users, each under MaxSeats=4.
	e.mustHold(e.users[0], "A1", "A2", "A3", "A4")
	held := e.mustHold(e.users[2], "B1", "B2", "B3", "B4")
	e.mustHold(e.users[3], "B5")

	entry, err := svc.Join(e.ctx, e.users[1], e.showID)
	e.must(err)
	if entry.Status != models.WaitlistWaiting {
		t.Fatalf("status = %s, want waiting", entry.Status)
	}

	// Force one of user2's held seats to expire, then run the sweep.
	e.shiftHold(held.BookingID, -1)
	_, err = e.svc.SweepExpired(e.ctx, 100)
	e.must(err)

	entries, err := svc.MyEntries(e.ctx, e.users[1])
	e.must(err)
	if len(entries) != 1 || entries[0].Status != models.WaitlistNotified || entries[0].FulfilledBookingID == "" {
		t.Fatalf("entries = %+v, want 1 notified entry with a fulfilled booking", entries)
	}

	b := e.booking(entries[0].FulfilledBookingID)
	if b.UserID != e.users[1] || b.Status != models.BookingPending {
		t.Fatalf("fulfilled booking = %+v, want pending booking for users[1]", b)
	}
}
