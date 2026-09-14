package service_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// ---------------------------------------------------------------------------
// Hold (F7)
// ---------------------------------------------------------------------------

// T1: many users race for one seat, exactly one wins.
func TestHold_ConcurrentSameSeat_OneWins(t *testing.T) {
	e := newEnv(t)
	var wg sync.WaitGroup
	var won, conflicts atomic.Int32
	for _, u := range e.users {
		wg.Add(1)
		go func(user string) {
			defer wg.Done()
			_, err := e.hold(user, "A1")
			switch {
			case err == nil:
				won.Add(1)
			case httpStatus(err) == http.StatusConflict:
				conflicts.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}(u)
	}
	wg.Wait()
	if won.Load() != 1 || int(conflicts.Load()) != len(e.users)-1 {
		t.Fatalf("won=%d conflicts=%d, want 1/%d", won.Load(), conflicts.Load(), len(e.users)-1)
	}
	e.wantSeat("A1", models.SeatStatusHeld)
	if n := e.count(`SELECT COUNT(*) FROM bookings`); n != 1 {
		t.Fatalf("bookings = %d, want 1", n)
	}
	e.checkInvariants()
}

// T2: a lot containing a taken seat rolls back entirely and names the seat.
func TestHold_LotWithTakenSeat_AllOrNothing(t *testing.T) {
	e := newEnv(t)
	e.mustHold(e.users[0], "A2")

	_, err := e.hold(e.users[1], "A1", "A2", "A3")
	if httpStatus(err) != http.StatusConflict {
		t.Fatalf("err = %v, want 409", err)
	}
	if got := apperrors.From(err).Details["seats"]; got != "A2" {
		t.Fatalf("reported seats = %q, want A2", got)
	}
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.wantSeat("A3", models.SeatStatusAvailable)
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE user_id = ?`, e.users[1]); n != 0 {
		t.Fatalf("user 1 bookings = %d, want 0", n)
	}
}

// T3: retries with the same idempotency key return the same booking, also
// when they arrive concurrently.
func TestHold_IdempotencyKey(t *testing.T) {
	e := newEnv(t)
	req := dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("A1", "A2"), IdempotencyKey: "key-1"}

	var wg sync.WaitGroup
	results := make([]string, 6)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := e.svc.Hold(e.ctx, e.users[0], req)
			if err != nil {
				t.Errorf("hold %d: %v", i, err)
				return
			}
			results[i] = res.BookingID
		}(i)
	}
	wg.Wait()
	for _, id := range results {
		if id != results[0] {
			t.Fatalf("booking ids differ: %v", results)
		}
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings`); n != 1 {
		t.Fatalf("bookings = %d, want 1", n)
	}

	_, err := e.svc.Hold(e.ctx, e.users[1], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("B1"), IdempotencyKey: "key-1"})
	if httpStatus(err) != http.StatusConflict {
		t.Fatalf("foreign key reuse err = %v, want 409", err)
	}
}

// T21 / E-HO10..12: a second tab replaces the first hold without extending
// its expiry; seats kept across both lots stay held.
func TestHold_SecondTabReplaces(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	first := e.mustHold(u, "A1", "A2")
	second := e.mustHold(u, "A2", "A3")

	if second.ReplacedBookingID != first.BookingID {
		t.Fatalf("replaced = %q, want %q", second.ReplacedBookingID, first.BookingID)
	}
	if d := second.ExpiresAt.Sub(first.ExpiresAt); d < -time.Millisecond || d > time.Millisecond {
		t.Fatalf("expiry moved by %v (E-HO11)", d)
	}
	old := e.wantStatus(first.BookingID, models.BookingExpired)
	e.wantReason(old, models.ReasonReplaced)
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.wantSeat("A2", models.SeatStatusHeld)
	e.wantSeat("A3", models.SeatStatusHeld)
	if second.TotalAmount != 2*priceStandard {
		t.Fatalf("total = %d", second.TotalAmount)
	}

	var wg sync.WaitGroup
	for _, lot := range [][]string{{"B1"}, {"B2", "B3"}} {
		wg.Add(1)
		go func(lot []string) {
			defer wg.Done()
			if _, err := e.hold(e.users[1], lot...); err != nil {
				t.Errorf("hold %v: %v", lot, err)
			}
		}(lot)
	}
	wg.Wait()
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE user_id = ? AND status = 'pending'`, e.users[1]); n != 1 {
		t.Fatalf("pending bookings = %d, want 1", n)
	}
	held := e.count(`SELECT COUNT(*) FROM showtime_seats WHERE held_by = ? AND status = 'held'`, e.users[1])
	seats := e.count(`SELECT COUNT(*) FROM booking_seats bs JOIN bookings b ON b.id = bs.booking_id
		WHERE b.user_id = ? AND b.status = 'pending'`, e.users[1])
	if held != seats {
		t.Fatalf("held seats %d != pending booking seats %d", held, seats)
	}
	e.checkInvariants()
}

// Replacing a booking whose checkout is open would drop a payment in flight.
func TestHold_BlockedWhilePaymentInProgress(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	e.pay(e.users[0], h.BookingID)

	_, err := e.hold(e.users[0], "A2")
	if !isAppErr(err, apperrors.ErrPaymentInProgress) {
		t.Fatalf("err = %v, want ErrPaymentInProgress", err)
	}
	e.wantStatus(h.BookingID, models.BookingPending)
	e.wantSeat("A1", models.SeatStatusHeld)
}

// E-HO5: a hold that expired by DB clock is taken over before the sweep runs.
func TestHold_TakesOverExpiredHold(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	e.shiftHold(h.BookingID, -1)

	if _, err := e.hold(e.users[1], "A1"); err != nil {
		t.Fatalf("take over expired hold: %v", err)
	}
	if got := e.seatState("A1").HeldBy; got == nil || *got != e.users[1] {
		t.Fatalf("A1 held by %v, want user 1", got)
	}
}

// E-HO6, E-HO7, gaps, foreign seats.
func TestHold_Rejections(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]

	if _, err := e.hold(u, "A1", "A2", "A3", "A4", "B1"); httpStatus(err) != http.StatusBadRequest {
		t.Errorf("over seat limit: err = %v, want 400", err)
	}
	if _, err := e.hold(u, "A5"); !isAppErr(err, apperrors.ErrSeatNotSellable) {
		t.Errorf("gap seat: err = %v, want ErrSeatNotSellable", err)
	}
	other := e.newShowtime(8 * time.Hour)
	foreign := e.seatsOf(other)["A1"]
	if _, err := e.svc.Hold(e.ctx, u, dto.HoldRequest{ShowID: e.showID, SeatIDs: []string{foreign}}); httpStatus(err) != http.StatusBadRequest {
		t.Errorf("seat of another show: err = %v, want 400", err)
	}

	e.must(e.db.Exec(`UPDATE showtimes SET status = 'closed' WHERE id = ?`, e.showID).Error)
	if _, err := e.hold(u, "A1"); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Errorf("closed show: err = %v, want ErrShowtimeClosed", err)
	}
	e.must(e.db.Exec(`UPDATE showtimes SET status = 'open', start_at = NOW() - INTERVAL '1 minute' WHERE id = ?`, e.showID).Error)
	if _, err := e.hold(u, "A1"); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Errorf("started show: err = %v, want ErrShowtimeClosed", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings`); n != 0 {
		t.Fatalf("bookings = %d, want 0", n)
	}
}

// ---------------------------------------------------------------------------
// Pay / IPN / confirm (F8, F9)
// ---------------------------------------------------------------------------

// T10: hold 3 seats -> pay -> IPN -> 3 tickets with locked prices; realtime
// viewers see held then sold; a ticket email job is published.
func TestPayConfirm_HappyPath(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	sub, err := e.hub.Subscribe(e.showID, "viewer")
	e.must(err)
	defer sub.Close()

	h := e.mustHold(u, "A1", "B1", "B2")
	if want := int64(priceStandard + 2*priceVIP); h.TotalAmount != want {
		t.Fatalf("total = %d, want %d", h.TotalAmount, want)
	}
	expectEvent(t, sub.Events, models.SeatStatusHeld, 3)

	// Prices changing after hold do not touch the booking (E-HO9).
	e.must(e.db.Exec(`UPDATE hall_prices SET price = price * 2 WHERE hall_id = ?`, e.hallID).Error)

	pay, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock", ClientIP: "203.0.113.7"})
	e.must(err)
	if !strings.HasPrefix(pay.RedirectURL, merchantURL+"/mock-gateway/checkout?ref=") {
		t.Fatalf("redirect = %s", pay.RedirectURL)
	}
	again, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	if again.TxnRef != pay.TxnRef || again.PaymentID != pay.PaymentID {
		t.Fatalf("second pay opened a new attempt (E-P11)")
	}
	if txn, _ := e.gw.Lookup(pay.TxnRef); txn.NotifyURL != merchantURL+"/api/v1/payments/mock/ipn" || txn.Amount != h.TotalAmount {
		t.Fatalf("gateway got notify %q amount %d", txn.NotifyURL, txn.Amount)
	}

	e.capture(pay.TxnRef)
	if ack := e.ipn(pay.TxnRef); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	expectEvent(t, sub.Events, models.SeatStatusSold, 3)

	order, err := e.svc.Order(e.ctx, u, h.BookingID)
	e.must(err)
	if order.Payment == nil || order.Payment.Status != models.PaymentPaid || order.Payment.Provider != "mock" {
		t.Fatalf("payment summary = %+v", order.Payment)
	}
	if len(order.Tickets) != 3 {
		t.Fatalf("tickets = %d, want 3", len(order.Tickets))
	}
	var sum int64
	codes := map[string]bool{}
	for _, tk := range order.Tickets {
		sum += tk.Price
		if len(tk.Code) != 32 || codes[tk.Code] {
			t.Fatalf("bad ticket code %q", tk.Code)
		}
		codes[tk.Code] = true
	}
	if sum != h.TotalAmount {
		t.Fatalf("ticket prices sum %d, want %d", sum, h.TotalAmount)
	}
	for _, l := range []string{"A1", "B1", "B2"} {
		e.wantSeat(l, models.SeatStatusSold)
	}
	if e.pub.count() != 1 {
		t.Fatalf("email jobs published = %d, want 1", e.pub.count())
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = 'orders.confirm'`, h.BookingID); n != 1 {
		t.Fatalf("confirm audit rows = %d, want 1", n)
	}
	e.checkInvariants()
}

// expectEvent waits for one realtime batch of n seats all in status.
func expectEvent(t *testing.T, events <-chan sse.SeatEvent, status string, n int) {
	t.Helper()
	select {
	case ev := <-events:
		if len(ev.Seats) != n {
			t.Fatalf("event has %d seats, want %d", len(ev.Seats), n)
		}
		for _, s := range ev.Seats {
			if s.Status != status {
				t.Fatalf("event seat %s status %s, want %s", s.ID, s.Status, status)
			}
		}
	case <-time.After(time.Second):
		t.Fatalf("no %s event broadcast", status)
	}
}

// T6 / E-P1: forged callback is rejected, audited and changes nothing.
func TestIPN_InvalidSignature(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref)

	if ack := e.notify(e.mockP, forgedIPN(ref, h.TotalAmount)); ack != payment.AckInvalidSignature {
		t.Fatalf("ack = %s, want invalid signature", ack)
	}
	if b := e.wantStatus(h.BookingID, models.BookingPending); b.PaidAt != nil {
		t.Fatal("forged IPN marked the booking paid")
	}
	if p := e.payment(ref); p.Status != models.PaymentPending {
		t.Fatalf("payment status = %s", p.Status)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE action = 'payments.notify' AND outcome = 'failure'`); n != 1 {
		t.Fatalf("rejection audit rows = %d, want 1", n)
	}
}

// T7 / E-P2: the same callback delivered 3 times concurrently applies once.
func TestIPN_DuplicateCallbacks(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1", "A2")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref)

	acks := make([]payment.AckStatus, 3)
	var wg sync.WaitGroup
	for i := range acks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			acks[i] = e.ipn(ref)
		}(i)
	}
	wg.Wait()
	processed := 0
	for _, a := range acks {
		switch a {
		case payment.AckProcessed:
			processed++
		case payment.AckDuplicate:
		default:
			t.Fatalf("ack = %s", a)
		}
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1 (%v)", processed, acks)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	if n := e.count(`SELECT COUNT(*) FROM tickets WHERE booking_id = ?`, h.BookingID); n != 2 {
		t.Fatalf("tickets = %d, want 2", n)
	}
	if e.gw.Refunds() != 0 {
		t.Fatalf("refunds = %d, want 0", e.gw.Refunds())
	}
	e.checkInvariants()
}

// E-C3 regression: IPN, reconcile and customer confirm racing each other must
// converge on CONFIRMED — never refund a sold booking.
func TestConfirm_ConcurrentPathsNeverRefundASale(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1", "A2")
	ref := e.pay(u, h.BookingID)
	e.capture(ref)

	ipn := func() error {
		if ack := e.ipn(ref); ack != payment.AckProcessed && ack != payment.AckDuplicate {
			return fmt.Errorf("ack %s", ack)
		}
		return nil
	}
	calls := []func() error{
		ipn,
		func() error { _, err := e.svc.Status(e.ctx, u, h.BookingID); return err },
		func() error { _, err := e.svc.Confirm(e.ctx, u, h.BookingID); return err },
		func() error { _, err := e.svc.Order(e.ctx, u, h.BookingID); return err },
		ipn,
		func() error { _, err := e.svc.Confirm(e.ctx, u, h.BookingID); return err },
	}
	var wg sync.WaitGroup
	for _, call := range calls {
		wg.Add(1)
		go func(call func() error) {
			defer wg.Done()
			if err := call(); err != nil {
				t.Errorf("call: %v", err)
			}
		}(call)
	}
	wg.Wait()
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	if e.gw.Refunds() != 0 {
		t.Fatalf("refunds = %d, want 0", e.gw.Refunds())
	}
	e.checkInvariants()
}

// T8 / E-P4: the gateway settles a different amount -> no sale, refund of
// what was actually collected.
func TestIPN_AmountMismatch_Refunds(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref, mock.CaptureOptions{AmountDelta: 1000})

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	b := e.wantStatus(h.BookingID, models.BookingRefunded)
	e.wantReason(b, models.ReasonAmountMismatch)
	assertRefunded(t, e, h.BookingID, ref)
	if p := e.payment(ref); p.PaidAmount == nil || *p.PaidAmount != h.TotalAmount+1000 {
		t.Fatalf("paid amount = %v", p.PaidAmount)
	}
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.checkInvariants()
}

// T4 / E-C1: payment lands just after the hold expired (sweep not run yet).
func TestConfirm_AfterHoldExpired_Refunds(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1", "A2")
	ref := e.pay(e.users[0], h.BookingID)
	e.shiftHold(h.BookingID, -0.5)
	e.capture(ref)

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantReason(e.booking(h.BookingID), models.ReasonHoldExpired)
	assertRefunded(t, e, h.BookingID, ref)
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.wantSeat("A2", models.SeatStatusAvailable)
	e.checkInvariants()
}

// T22: the seat went to another customer before the payment arrived.
func TestConfirm_SeatTakenOver_RefundsAndKeepsWinner(t *testing.T) {
	e := newEnv(t)
	loser, winner := e.users[0], e.users[1]
	h := e.mustHold(loser, "A1")
	ref := e.pay(loser, h.BookingID)
	e.shiftHold(h.BookingID, -1)
	w := e.mustHold(winner, "A1")
	e.capture(ref)

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	assertRefunded(t, e, h.BookingID, ref)
	e.wantStatus(w.BookingID, models.BookingPending)
	if got := e.seatState("A1").HeldBy; got == nil || *got != winner {
		t.Fatalf("A1 held by %v, want winner", got)
	}
	e.checkInvariants()
}

// E-C2: the fencing version moved under an unexpired booking.
func TestConfirm_FencingVersionMismatch_Refunds(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1", "A2")
	ref := e.pay(e.users[0], h.BookingID)
	e.must(e.db.Exec(`UPDATE showtime_seats SET version = version + 1 WHERE id = ?`, e.seat["A2"]).Error)
	e.capture(ref)

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantReason(e.booking(h.BookingID), models.ReasonSeatsLost)
	assertRefunded(t, e, h.BookingID, ref)
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.checkInvariants()
}

// E-C5: the show was closed between payment and confirm.
func TestConfirm_ShowtimeClosed_Refunds(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.must(e.db.Exec(`UPDATE showtimes SET status = 'closed' WHERE id = ?`, e.showID).Error)
	e.capture(ref)

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantReason(e.booking(h.BookingID), models.ReasonShowtimeClosed)
	assertRefunded(t, e, h.BookingID, ref)
}

// T5 / E-R1: sweep and confirm race on the TTL boundary. Whoever wins, no seat
// is both released and sold, and money is refunded exactly when not sold.
func TestSweepVsConfirm_Race(t *testing.T) {
	for i := 0; i < 8; i++ {
		e := newEnv(t)
		u := e.users[0]
		h := e.mustHold(u, "A1", "A2")
		ref := e.pay(u, h.BookingID)
		e.capture(ref)
		req := e.notificationRequest(e.mockP, ref)
		e.shiftHold(h.BookingID, 0.08)
		time.Sleep(time.Duration(70+i*2) * time.Millisecond)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if ack := e.notify(e.mockP, req); ack != payment.AckProcessed {
				t.Errorf("ack = %s", ack)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
					t.Errorf("sweep: %v", err)
				}
				time.Sleep(5 * time.Millisecond)
			}
		}()
		wg.Wait()
		if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
			t.Fatal(err)
		}

		b := e.booking(h.BookingID)
		sold := e.count(`SELECT COUNT(*) FROM showtime_seats WHERE status = 'sold'`)
		switch b.Status {
		case models.BookingConfirmed:
			if sold != 2 || e.gw.Refunds() != 0 {
				t.Fatalf("run %d confirmed: sold=%d refunds=%d", i, sold, e.gw.Refunds())
			}
		case models.BookingRefunded:
			if sold != 0 || e.gw.Refunds() != 1 {
				t.Fatalf("run %d refunded: sold=%d refunds=%d", i, sold, e.gw.Refunds())
			}
		default:
			t.Fatalf("run %d: status %s", i, b.Status)
		}
		e.checkInvariants()
	}
}

// E-P10: gateway down -> 502, booking untouched, the failed attempt is kept
// and a retry opens a new one.
func TestPay_GatewayDown(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	e.gw.SetDown(true)
	if _, err := e.svc.Pay(e.ctx, e.users[0], h.BookingID, dto.PayRequest{Provider: "mock"}); httpStatus(err) != http.StatusBadGateway {
		t.Fatalf("err = %v, want 502", err)
	}
	e.wantStatus(h.BookingID, models.BookingPending)
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ? AND status = 'failed' AND status_reason = 'create_failed'`, h.BookingID); n != 1 {
		t.Fatalf("failed attempts = %d, want 1", n)
	}
	e.gw.SetDown(false)
	e.pay(e.users[0], h.BookingID)
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ?`, h.BookingID); n != 2 {
		t.Fatalf("attempts = %d, want 2", n)
	}
}

// E-P5: the IPN never arrives; reading the order reconciles with the provider.
func TestReconcile_LostIPN(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	e.capture(e.pay(e.users[0], h.BookingID))

	st, err := e.svc.Status(e.ctx, e.users[0], h.BookingID)
	e.must(err)
	if st.Status != models.BookingConfirmed || st.Payment == nil || st.Payment.Status != models.PaymentPaid {
		t.Fatalf("status = %s payment = %+v", st.Status, st.Payment)
	}
	e.checkInvariants()
}

// E-O1: another customer's order is forbidden, not hidden.
func TestOrder_OtherUserForbidden(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1")
	if _, err := e.svc.Order(e.ctx, e.users[1], id); httpStatus(err) != http.StatusForbidden {
		t.Fatalf("err = %v, want 403", err)
	}
}

// ---------------------------------------------------------------------------
// Sweep (F10)
// ---------------------------------------------------------------------------

// E-P7 / E-O2: unpaid overdue bookings expire, their seats free up, the order
// stays in history and can not be paid anymore.
func TestSweep_ExpiresUnpaidBookings(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1", "A2")
	e.shiftHold(h.BookingID, -1)

	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.ReleasedSeats != 2 || res.ExpiredBookings != 1 {
		t.Fatalf("sweep = %+v", res)
	}
	b := e.wantStatus(h.BookingID, models.BookingExpired)
	e.wantReason(b, models.ReasonHoldExpired)
	e.wantSeat("A1", models.SeatStatusAvailable)

	if _, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{}); !isAppErr(err, apperrors.ErrBookingExpired) {
		t.Fatalf("pay expired: err = %v", err)
	}
	items, _, err := e.svc.List(e.ctx, u, dto.PageQuery{Page: 1, PageSize: 10})
	e.must(err)
	if len(items) != 1 || items[0].Status != models.BookingExpired {
		t.Fatalf("history = %+v", items)
	}
}

// E-P3: money arrives for a booking the sweep already expired.
func TestIPN_PaidAfterExpiry_Refunds(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.shiftHold(h.BookingID, -1)
	_, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	e.wantStatus(h.BookingID, models.BookingExpired)
	e.capture(ref)

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantReason(e.booking(h.BookingID), models.ReasonPaidAfterExpiry)
	assertRefunded(t, e, h.BookingID, ref)
	e.checkInvariants()
}

// Crash recovery: payment recorded but confirm never ran -> sweep confirms.
func TestSweep_FinalizesStuckPaidBooking(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref)
	e.must(e.db.Exec(`WITH p AS (
			UPDATE payments SET status = 'paid', paid_amount = amount, paid_at = NOW() - INTERVAL '2 minutes'
			WHERE txn_ref = ? RETURNING id, booking_id)
		UPDATE bookings b SET paid_at = NOW() - INTERVAL '2 minutes', payment_id = p.id
		FROM p WHERE b.id = p.booking_id`, ref).Error)

	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.FinalizedPaid != 1 {
		t.Fatalf("sweep = %+v", res)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	e.checkInvariants()
}

// ---------------------------------------------------------------------------
// Gate (F12)
// ---------------------------------------------------------------------------

// T11 / E-T1 / E-T2.
func TestRedeem(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1", "A2")
	order, err := e.svc.Order(e.ctx, e.users[0], id)
	e.must(err)
	first, second := order.Tickets[0], order.Tickets[1]
	otherShow := e.newShowtime(8 * time.Hour)
	e.moveShowStart(e.showID, 10*time.Minute) // inside the check-in window (E-T4)

	check := func(ref, show, want string) *dto.RedeemResponse {
		t.Helper()
		res, err := e.svc.Redeem(e.ctx, ref, show)
		e.must(err)
		if res.Status != want {
			t.Fatalf("redeem %s at %s = %s, want %s", ref, show, res.Status, want)
		}
		return res
	}

	res := check(first.Code, e.showID, models.RedeemOK)
	if res.SeatLabel != first.SeatLabel || res.HallName != "Hall 1" {
		t.Fatalf("verdict details = %+v", res)
	}
	check(strings.ToLower(first.Code), e.showID, models.RedeemUsed)
	check(second.ID, otherShow, models.RedeemWrongShow)
	check(second.ID, e.showID, models.RedeemOK) // wrong_show did not consume it
	check("DOES-NOT-EXIST", e.showID, models.RedeemNotFound)

	id2 := e.confirmed(e.users[1], "B1")
	o2, err := e.svc.Order(e.ctx, e.users[1], id2)
	e.must(err)
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := e.svc.Redeem(context.Background(), o2.Tickets[0].Code, e.showID)
			if err == nil && r.Status == models.RedeemOK {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("concurrent redeem OK = %d, want 1", ok.Load())
	}
}

// assertRefunded checks the whole money trail of a refunded booking paid
// through the mock provider.
func assertRefunded(t *testing.T, e *env, bookingID, ref string) {
	t.Helper()
	b := e.wantStatus(bookingID, models.BookingRefunded)
	p := e.payment(ref)
	if b.PaidAt == nil || b.PaymentID == nil || *b.PaymentID != p.ID {
		t.Fatalf("booking paid_at=%v payment_id=%v, want payment %s", b.PaidAt, b.PaymentID, p.ID)
	}
	if p.Status != models.PaymentRefunded || p.RefundedAt == nil {
		t.Fatalf("payment status=%s refunded_at=%v", p.Status, p.RefundedAt)
	}
	if txn, _ := e.gw.Lookup(ref); txn.State != mock.TxnRefunded {
		t.Fatalf("gateway state = %s, want refunded", txn.State)
	}
	if n := e.count(`SELECT COUNT(*) FROM tickets WHERE booking_id = ?`, bookingID); n != 0 {
		t.Fatalf("tickets on refunded booking = %d", n)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = 'orders.refund'`, bookingID); n != 1 {
		t.Fatalf("refund audit rows = %d, want 1", n)
	}
}
