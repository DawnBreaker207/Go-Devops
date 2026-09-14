package service_test

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
)

func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// Random overlapping holds, expiring holds, late payments and a non-stop sweep never double-sell a seat.
func TestStress_NoSeatSoldTwice(t *testing.T) {
	labels := []string{"A1", "A2", "A3", "A4", "B1", "B2", "B3", "B4", "B5"}
	const rounds, attempts = 10, 15

	for round := 0; round < rounds; round++ {
		e := newEnv(t)
		var holds, conflicts, paid, late atomic.Int32
		stop := make(chan struct{})
		sweepDone := make(chan struct{})
		go func() {
			defer close(sweepDone)
			for {
				select {
				case <-stop:
					return
				default:
					if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
						t.Errorf("sweep: %v", err)
					}
				}
			}
		}()

		var wg sync.WaitGroup
		for _, user := range e.users {
			wg.Add(1)
			go func(user string) {
				defer wg.Done()
				for i := 0; i < attempts; i++ {
					perm := rand.Perm(len(labels))
					lot := make([]string, 1+rand.IntN(3))
					for j := range lot {
						lot[j] = labels[perm[j]]
					}
					h, err := e.hold(user, lot...)
					if err != nil {
						if httpStatus(err) == http.StatusConflict {
							conflicts.Add(1)
							continue
						}
						t.Errorf("hold %v: %v", lot, err)
						return
					}
					holds.Add(1)
					if rand.IntN(2) == 0 {
						// walk away; half the time the hold runs out so the sweep frees it
						if rand.IntN(2) == 0 {
							e.shiftHold(h.BookingID, -1)
						}
						continue
					}
					res, err := e.svc.Pay(e.ctx, user, h.BookingID, dto.PayRequest{Provider: "mock"})
					if err != nil {
						if httpStatus(err) != http.StatusConflict {
							t.Errorf("pay: %v", err)
						}
						continue
					}
					e.capture(res.TxnRef)
					if rand.IntN(2) == 0 {
						// paid after the hold expired: the seat may be gone and need a refund
						e.shiftHold(h.BookingID, -1)
						late.Add(1)
					}
					e.ipn(res.TxnRef)
					paid.Add(1)
				}
			}(user)
		}
		wg.Wait()
		close(stop)
		<-sweepDone

		// An expired hold may read HELD until the next sweep; only running holds must have their booking.
		orphanHeld := `SELECT COUNT(*) FROM showtime_seats ss WHERE ss.status = 'held' AND %s AND (
			SELECT COUNT(*) FROM booking_seats bs JOIN bookings b ON b.id = bs.booking_id
			WHERE bs.showtime_seat_id = ss.id AND b.status = 'pending' AND b.user_id = ss.held_by AND bs.hold_version = ss.version) <> 1`
		if n := e.count(fmt.Sprintf(orphanHeld, "ss.held_until >= NOW()")); n != 0 {
			t.Errorf("round %d: %d seats held by a running hold without its pending booking", round, n)
		}
		staleHeld := e.count(fmt.Sprintf(orphanHeld, "ss.held_until < NOW()"))
		if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
			t.Fatal(err)
		}

		sold := e.count(`SELECT COUNT(*) FROM showtime_seats WHERE status = 'sold'`)
		refunded := e.count(`SELECT COUNT(*) FROM bookings WHERE status = 'refunded'`)
		t.Logf("round %d: holds=%d conflicts=%d paid=%d (late %d) sold=%d refunded=%d stale-held-before-sweep=%d",
			round, holds.Load(), conflicts.Load(), paid.Load(), late.Load(), sold, refunded, staleHeld)

		checks := map[string]string{
			"two tickets on one seat": `SELECT COUNT(*) FROM (SELECT 1 FROM tickets GROUP BY showtime_seat_id HAVING COUNT(*) > 1) x`,
			"sold seats <> tickets of confirmed bookings": `SELECT ABS((SELECT COUNT(*) FROM showtime_seats WHERE status = 'sold') -
				(SELECT COUNT(*) FROM tickets t JOIN bookings b ON b.id = t.booking_id WHERE b.status = 'confirmed'))`,
			"held seat not owned by exactly one pending booking": `SELECT COUNT(*) FROM showtime_seats ss WHERE ss.status = 'held' AND (
				SELECT COUNT(*) FROM booking_seats bs JOIN bookings b ON b.id = bs.booking_id
				WHERE bs.showtime_seat_id = ss.id AND b.status = 'pending' AND b.user_id = ss.held_by AND bs.hold_version = ss.version) <> 1`,
			"seat in two live bookings": `SELECT COUNT(*) FROM (SELECT bs.showtime_seat_id FROM booking_seats bs
				JOIN bookings b ON b.id = bs.booking_id JOIN showtime_seats ss ON ss.id = bs.showtime_seat_id
				WHERE b.status = 'confirmed' OR (b.status = 'pending' AND ss.status = 'held' AND bs.hold_version = ss.version)
				GROUP BY bs.showtime_seat_id HAVING COUNT(*) > 1) x`,
			"paid money neither confirmed nor refunded": `SELECT COUNT(*) FROM payments p JOIN bookings b ON b.id = p.booking_id
				WHERE p.status = 'paid' AND b.status <> 'confirmed'`,
		}
		for name, sql := range checks {
			if n := e.count(sql); n != 0 {
				t.Errorf("round %d: %s (%d)", round, name, n)
			}
		}
		var odd []struct {
			Label, Status                      string
			Expired                            bool
			Version                            int64
			BookingStatus, Reason              *string
			HoldVersion                        *int64
			SameUser, BookingOverdue, PaidFlag *bool
		}
		e.must(e.db.Raw(`SELECT s.row_label || s.col_number AS label, ss.status, ss.held_until < NOW() AS expired, ss.version,
				b.status AS booking_status, b.status_reason AS reason, bs.hold_version,
				b.user_id = ss.held_by AS same_user, b.expires_at < NOW() AS booking_overdue, b.paid_at IS NOT NULL AS paid_flag
			FROM showtime_seats ss JOIN seats s ON s.id = ss.seat_id
			LEFT JOIN booking_seats bs ON bs.showtime_seat_id = ss.id
			LEFT JOIN bookings b ON b.id = bs.booking_id
			WHERE ss.status = 'held' AND (SELECT COUNT(*) FROM booking_seats bs2 JOIN bookings b2 ON b2.id = bs2.booking_id
				WHERE bs2.showtime_seat_id = ss.id AND b2.status = 'pending' AND b2.user_id = ss.held_by AND bs2.hold_version = ss.version) <> 1
			ORDER BY label, bs.hold_version`).Scan(&odd).Error)
		for _, r := range odd {
			t.Logf("  odd seat %s status=%s expired=%v v=%d | booking=%v reason=%v hold_v=%v same_user=%v overdue=%v paid=%v",
				r.Label, r.Status, r.Expired, r.Version, deref(r.BookingStatus), deref(r.Reason), deref(r.HoldVersion),
				deref(r.SameUser), deref(r.BookingOverdue), deref(r.PaidFlag))
		}
		e.checkInvariants()
	}
}
