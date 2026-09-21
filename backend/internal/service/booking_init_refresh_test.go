package service_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// Init opens a seatless PENDING booking with a TTL, idempotent on re-entry.
func TestInit_OpensSeatlessBooking(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]

	res, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)
	if res.BookingID == "" || res.ShowtimeID != e.showID {
		t.Fatalf("init = %+v, want booking for show %s", res, e.showID)
	}
	if res.Reused {
		t.Fatalf("first init reused = true")
	}
	if res.TTLSeconds != int64((10 * time.Minute).Seconds()) {
		t.Fatalf("ttl = %d, want 600", res.TTLSeconds)
	}
	b := e.wantStatus(res.BookingID, models.BookingPending)
	if b.TotalAmount != 0 {
		t.Fatalf("total = %d, want 0", b.TotalAmount)
	}
	if b.ExpiresAt == nil || time.Until(*b.ExpiresAt) < 9*time.Minute {
		t.Fatalf("expires_at = %v, want ~+10m", b.ExpiresAt)
	}
	if n := e.count(`SELECT COUNT(*) FROM booking_seats WHERE booking_id = ?`, res.BookingID); n != 0 {
		t.Fatalf("booking seats = %d, want 0", n)
	}
	e.checkInvariants()
}

func TestInit_ReentryReturnsSameBooking(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]

	first, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)
	second, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)
	if second.BookingID != first.BookingID || !second.Reused {
		t.Fatalf("re-entry = %+v, want same booking reused", second)
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE user_id = ?`, u); n != 1 {
		t.Fatalf("bookings = %d, want 1", n)
	}
}

func TestInit_Validation(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]

	if _, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: "00000000-0000-0000-0000-000000000000"}); httpStatus(err) != 404 {
		t.Fatalf("unknown show err = %v, want 404", err)
	}
	e.moveShowStart(e.showID, -time.Hour)
	if _, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID}); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("started show err = %v, want showtime closed", err)
	}
}

func TestInit_WithExistingHoldReturnsIt(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	res, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)
	if res.BookingID != h.BookingID || !res.Reused {
		t.Fatalf("init with hold = %+v, want hold %s reused", res, h.BookingID)
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE user_id = ?`, u); n != 1 {
		t.Fatalf("bookings = %d, want 1 (init adds no row)", n)
	}
}

func TestInit_PaidBookingInProgressRefused(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")
	e.pay(u, h.BookingID) // open attempt, no IPN yet

	if _, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID}); !isAppErr(err, apperrors.ErrPaymentInProgress) {
		t.Fatalf("init with open payment err = %v, want payment in progress", err)
	}
}

// Refresh extends a pending hold inside the lifetime cap.
func TestRefresh_ExtendsWithinCap(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	res, err := e.svc.Refresh(e.ctx, u, h.BookingID)
	e.must(err)
	if res.BookingID != h.BookingID {
		t.Fatalf("refresh booking = %s, want %s", res.BookingID, h.BookingID)
	}
	if !res.ExpiresAt.After(h.ExpiresAt) {
		t.Fatalf("expiry %v not after %v", res.ExpiresAt, h.ExpiresAt)
	}
	e.wantStatus(h.BookingID, models.BookingPending)
	e.wantSeat("A1", models.SeatStatusHeld)
	e.checkInvariants()
}

func TestRefresh_CapRefused(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	// Age the booking to 25 of the 30 test minutes: now+10m would pass created+30m.
	e.must(e.db.Exec(`UPDATE bookings SET created_at = NOW() - make_interval(secs => 1500) WHERE id = ?`, h.BookingID).Error)
	if _, err := e.svc.Refresh(e.ctx, u, h.BookingID); !isAppErr(err, apperrors.ErrHoldLifetimeExceeded) {
		t.Fatalf("over-cap refresh err = %v, want lifetime exceeded", err)
	}
	if got := httpStatus(mustRefreshErr(t, e, u, h.BookingID)); got != 409 {
		t.Fatalf("status = %d, want 409", got)
	}
}

func mustRefreshErr(t *testing.T, e *env, user, bookingID string) error {
	t.Helper()
	_, err := e.svc.Refresh(e.ctx, user, bookingID)
	if err == nil {
		t.Fatalf("refresh unexpectedly succeeded")
	}
	return err
}

func TestRefresh_ExpiredPaidForeignAndMissing(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	e.shiftHold(h.BookingID, -660) // past the 10m TTL
	if _, err := e.svc.Refresh(e.ctx, u, h.BookingID); !isAppErr(err, apperrors.ErrBookingExpired) {
		t.Fatalf("expired refresh err = %v, want expired", err)
	}

	h2 := e.mustHold(u, "A2")
	e.must(e.db.Exec(`UPDATE bookings SET paid_at = NOW(), payment_id = '11111111-1111-1111-1111-111111111111' WHERE id = ?`, h2.BookingID).Error)
	if _, err := e.svc.Refresh(e.ctx, u, h2.BookingID); httpStatus(err) != 409 {
		t.Fatalf("paid refresh err = %v, want 409", err)
	}

	if _, err := e.svc.Refresh(e.ctx, e.users[1], h2.BookingID); httpStatus(err) != 403 {
		t.Fatalf("foreign refresh err = %v, want 403", err)
	}
	if _, err := e.svc.Refresh(e.ctx, u, "00000000-0000-0000-0000-000000000000"); httpStatus(err) != 404 {
		t.Fatalf("missing refresh err = %v, want 404", err)
	}
}

func TestRefresh_SeatLost(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	e.must(e.db.Exec(`UPDATE showtime_seats SET status = 'sold', held_by = NULL, held_until = NULL, version = version + 1
		WHERE id = (SELECT showtime_seat_id FROM booking_seats WHERE booking_id = ?)`, h.BookingID).Error)
	_, err := e.svc.Refresh(e.ctx, u, h.BookingID)
	if !isAppErr(err, apperrors.ErrSeatTaken) {
		t.Fatalf("lost-seat refresh err = %v, want seat taken", err)
	}
}

func TestRefresh_ConcurrentBothSucceed(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = e.svc.Refresh(e.ctx, u, h.BookingID)
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("concurrent refresh err = %v", err)
		}
	}
	e.wantStatus(h.BookingID, models.BookingPending)
	e.checkInvariants()
}

// Empty init shells can neither pay nor confirm.
func TestBookingEmpty_PayAndConfirmRefused(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	init, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)

	if _, err := e.svc.Pay(e.ctx, u, init.BookingID, dto.PayRequest{Provider: "mock"}); !isAppErr(err, apperrors.ErrBookingEmpty) {
		t.Fatalf("empty pay err = %v, want booking empty", err)
	}
	if _, err := e.svc.Confirm(e.ctx, u, init.BookingID); !isAppErr(err, apperrors.ErrBookingEmpty) {
		t.Fatalf("empty confirm err = %v, want booking empty", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ?`, init.BookingID); n != 0 {
		t.Fatalf("payments = %d, want 0 (no 0-amount attempt)", n)
	}
}

// Sweep expires seatless shells without touching seats.
func TestSweep_ExpiresSeatlessShell(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	init, err := e.svc.Init(e.ctx, u, dto.InitRequest{ShowID: e.showID})
	e.must(err)

	e.shiftHold(init.BookingID, -660)
	if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	b := e.wantStatus(init.BookingID, models.BookingExpired)
	e.wantReason(b, models.ReasonHoldExpired)
	e.checkInvariants()
}
