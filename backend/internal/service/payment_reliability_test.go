package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

var errGatewayBusy = errors.New("gateway busy")

func (e *env) refundPendingAfterMismatch(user, seat string) string {
	e.t.Helper()
	h := e.mustHold(user, seat)
	ref := e.pay(user, h.BookingID)
	e.capture(ref, mock.CaptureOptions{AmountDelta: 1000})
	if ack := e.ipn(ref); ack != payment.AckProcessed {
		e.t.Fatalf("ack = %s", ack)
	}
	if p := e.payment(ref); p.Status != models.PaymentRefundPending || p.RefundAttempts != 1 {
		e.t.Fatalf("payment = %s attempts=%d, want refund_pending after 1 try", p.Status, p.RefundAttempts)
	}
	return ref
}

func TestSweep_RefundsLateCaptureOfAbandonedAttempt(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")
	ref := e.pay(u, h.BookingID)

	e.shiftHold(h.BookingID, -1200)
	e.must(e.db.Exec(`UPDATE payments SET created_at = NOW() - INTERVAL '30 minutes',
		expires_at = NOW() - INTERVAL '20 minutes', checked_at = NULL WHERE txn_ref = ?`, ref).Error)
	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.AbandonedPayments != 1 || res.ExpiredBookings != 1 {
		t.Fatalf("first sweep = %+v", res)
	}

	e.capture(ref)
	res, err = e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.LateCaptures != 0 {
		t.Fatalf("attempt rechecked right after it was queried: %+v", res)
	}
	e.must(e.db.Exec(`UPDATE payments SET checked_at = NULL WHERE txn_ref = ?`, ref).Error)
	res, err = e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.LateCaptures != 1 {
		t.Fatalf("late capture sweep = %+v", res)
	}

	e.wantReason(e.wantStatus(h.BookingID, models.BookingRefunded), models.ReasonPaidAfterExpiry)
	if p := e.payment(ref); p.Status != models.PaymentRefunded {
		t.Fatalf("payment = %s, want refunded", p.Status)
	}
	if txn, _ := e.gw.Lookup(ref); txn.State != mock.TxnRefunded {
		t.Fatalf("gateway state = %s", txn.State)
	}
	e.checkInvariants()
}

func TestSweep_LateCaptureAfterOtherProviderPaid_RefundsDuplicate(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	other := e.addProvider("mock2")
	h := e.mustHold(u, "A1")
	refA := e.pay(u, h.BookingID)
	payB, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock2"})
	e.must(err)
	if _, err := other.Gateway().Capture(payB.TxnRef, mock.CaptureOptions{}); err != nil {
		t.Fatal(err)
	}
	if ack := e.ipnVia(other, payB.TxnRef); ack != payment.AckProcessed {
		t.Fatalf("mock2 ack = %s", ack)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)

	e.must(e.db.Exec(`UPDATE payments SET status = 'failed', status_reason = 'abandoned', checked_at = NULL
		WHERE txn_ref = ?`, refA).Error)
	e.capture(refA)
	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.LateCaptures != 1 {
		t.Fatalf("sweep = %+v", res)
	}
	if p := e.payment(refA); p.Status != models.PaymentRefunded || derefStr(p.StatusReason) != models.PaymentReasonDuplicate {
		t.Fatalf("late attempt = %s/%s, want refunded/duplicate", p.Status, derefStr(p.StatusReason))
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	e.checkInvariants()
}

func TestSweep_FailedAttemptOutsideWindowIgnored(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.must(e.db.Exec(`UPDATE payments SET status = 'failed', status_reason = 'abandoned', checked_at = NULL,
		created_at = NOW() - INTERVAL '26 hours', expires_at = NOW() - INTERVAL '25 hours' WHERE txn_ref = ?`, ref).Error)
	e.capture(ref)
	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.LateCaptures != 0 {
		t.Fatalf("sweep = %+v", res)
	}
	if p := e.payment(ref); p.Status != models.PaymentFailed {
		t.Fatalf("payment = %s, want failed", p.Status)
	}
}

func TestNotify_LocksBookingBeforePayment(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref)

	tx := e.db.Begin()
	defer tx.Rollback()
	e.must(tx.Exec(`SELECT 1 FROM bookings WHERE id = ? FOR UPDATE`, h.BookingID).Error)
	done := make(chan payment.AckStatus, 1)
	go func() { done <- e.ipn(ref) }()

	deadline := time.Now().Add(3 * time.Second)
	for e.count(`SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("notification never waited for the booking lock")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := tx.Exec(`SELECT 1 FROM payments WHERE txn_ref = ? FOR UPDATE NOWAIT`, ref).Error; err != nil {
		t.Fatalf("notification holds the payment lock while waiting for the booking: %v", err)
	}
	e.must(tx.Rollback().Error)
	if ack := <-done; ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
}

func TestRefund_ConcurrentSettleCallsProviderOnce(t *testing.T) {
	e := newEnv(t)
	e.gw.FailRefunds(errGatewayBusy)
	ref := e.refundPendingAfterMismatch(e.users[0], "A1")
	e.gw.FailRefunds(nil)
	e.gw.SetRefundDelay(300 * time.Millisecond)
	e.must(e.db.Exec(`UPDATE payments SET next_retry_at = NULL WHERE txn_ref = ?`, ref).Error)

	before := e.gw.RefundCalls()
	together(5, func(int) {
		if _, err := e.svc.SweepExpired(e.ctx, 500); err != nil {
			t.Errorf("sweep: %v", err)
		}
	})
	if calls := e.gw.RefundCalls() - before; calls != 1 {
		t.Fatalf("provider refund calls = %d, want 1", calls)
	}
	if p := e.payment(ref); p.Status != models.PaymentRefunded || e.gw.Refunds() != 1 {
		t.Fatalf("payment = %s, gateway refunds = %d", p.Status, e.gw.Refunds())
	}
	e.checkInvariants()
}

func TestSweep_RefundFailureBacksOff(t *testing.T) {
	e := newEnv(t)
	e.gw.FailRefunds(errGatewayBusy)
	ref := e.refundPendingAfterMismatch(e.users[0], "A1")
	p := e.payment(ref)
	if p.NextRetryAt == nil || !p.NextRetryAt.After(time.Now().Add(30*time.Second)) || !strings.Contains(derefStr(p.LastError), "gateway busy") {
		t.Fatalf("deferred refund: next_retry_at=%v last_error=%q", p.NextRetryAt, derefStr(p.LastError))
	}

	e.gw.FailRefunds(nil)
	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.SettledRefunds != 0 || e.payment(ref).Status != models.PaymentRefundPending {
		t.Fatalf("refund retried before its time: %+v", res)
	}
	e.must(e.db.Exec(`UPDATE payments SET next_retry_at = NOW() - INTERVAL '1 second' WHERE txn_ref = ?`, ref).Error)
	res, err = e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.SettledRefunds != 1 || e.payment(ref).Status != models.PaymentRefunded {
		t.Fatalf("due refund not settled: %+v", res)
	}
	e.checkInvariants()
}

func TestSweep_FailingRefundDoesNotStarveQueue(t *testing.T) {
	e := newEnv(t)
	other := e.addProvider("mock2")
	e.gw.FailRefunds(errGatewayBusy)
	stuckRef := e.refundPendingAfterMismatch(e.users[0], "A1")

	h := e.mustHold(e.users[1], "A2")
	payB, err := e.svc.Pay(e.ctx, e.users[1], h.BookingID, dto.PayRequest{Provider: "mock2"})
	e.must(err)
	other.Gateway().FailRefunds(errGatewayBusy)
	if _, err := other.Gateway().Capture(payB.TxnRef, mock.CaptureOptions{AmountDelta: 1000}); err != nil {
		t.Fatal(err)
	}
	if ack := e.ipnVia(other, payB.TxnRef); ack != payment.AckProcessed {
		t.Fatalf("mock2 ack = %s", ack)
	}
	other.Gateway().FailRefunds(nil)
	e.must(e.db.Exec(`UPDATE payments SET next_retry_at = NULL WHERE status = 'refund_pending'`).Error)

	for i := 0; i < 2; i++ {
		_, err := e.svc.SweepExpired(e.ctx, 1)
		e.must(err)
	}
	if p := e.payment(payB.TxnRef); p.Status != models.PaymentRefunded {
		t.Fatalf("refund behind a failing one = %s, want refunded", p.Status)
	}
	if p := e.payment(stuckRef); p.Status != models.PaymentRefundPending || p.RefundAttempts != 2 {
		t.Fatalf("failing refund = %s attempts=%d", p.Status, p.RefundAttempts)
	}
}

func TestSweep_StuckFinalizeBacksOff(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	p := e.payment(ref)
	e.must(e.db.Exec(`UPDATE payments SET status = 'paid', paid_amount = amount, paid_at = NOW() WHERE id = ?`, p.ID).Error)
	e.must(e.db.Exec(`UPDATE bookings SET paid_at = NOW() - INTERVAL '2 minutes', payment_id = ?,
		total_amount = total_amount + 1 WHERE id = ?`, p.ID, h.BookingID).Error)

	for i := 0; i < 2; i++ {
		res, err := e.svc.SweepExpired(e.ctx, 500)
		e.must(err)
		if res.FinalizedPaid != 0 {
			t.Fatalf("sweep %d = %+v", i, res)
		}
	}
	b := e.booking(h.BookingID)
	if b.FinalizeAttempts != 1 || b.NextFinalizeAt == nil || !b.NextFinalizeAt.After(time.Now()) {
		t.Fatalf("after 2 sweeps: attempts=%d next=%v, want 1 try and a future retry", b.FinalizeAttempts, b.NextFinalizeAt)
	}
	e.must(e.db.Exec(`UPDATE bookings SET next_finalize_at = NOW() - INTERVAL '1 second' WHERE id = ?`, h.BookingID).Error)
	_, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if b := e.booking(h.BookingID); b.FinalizeAttempts != 2 {
		t.Fatalf("due retry not run: attempts=%d", b.FinalizeAttempts)
	}
}

func TestPay_RefusedOnceShowtimeClosedOrStarted(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")

	e.must(e.db.Exec(`UPDATE showtimes SET status = 'closed' WHERE id = ?`, e.showID).Error)
	if _, err := e.svc.Pay(e.ctx, e.users[0], h.BookingID, dto.PayRequest{Provider: "mock"}); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("closed show: err = %v", err)
	}
	e.must(e.db.Exec(`UPDATE showtimes SET status = 'open', start_at = NOW() - INTERVAL '1 minute' WHERE id = ?`, e.showID).Error)
	if _, err := e.svc.Pay(e.ctx, e.users[0], h.BookingID, dto.PayRequest{Provider: "mock"}); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("started show: err = %v", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ?`, h.BookingID); n != 0 {
		t.Fatalf("payment attempts = %d, want 0", n)
	}
}

// An attempt without a checkout URL is resumed after the grace period; before that it answers 409.
func TestPay_ResumesOrphanCheckout(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")
	first, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)

	e.must(e.db.Exec(`UPDATE payments SET redirect_url = NULL WHERE id = ?`, first.PaymentID).Error)
	if _, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"}); httpStatus(err) != http.StatusConflict {
		t.Fatalf("fresh orphan: err = %v, want 409", err)
	}

	e.must(e.db.Exec(`UPDATE payments SET updated_at = NOW() - INTERVAL '1 minute' WHERE id = ?`, first.PaymentID).Error)
	again, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	if again.PaymentID != first.PaymentID || again.TxnRef != first.TxnRef || again.RedirectURL == "" {
		t.Fatalf("resumed = %+v, want the same attempt with a checkout URL", again)
	}
	if p := e.payment(first.TxnRef); p.RedirectURL == nil {
		t.Fatal("checkout URL not stored")
	}
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ?`, h.BookingID); n != 1 {
		t.Fatalf("attempts = %d, want 1", n)
	}
}

func TestHold_OrphanCheckoutDoesNotBlockNewHold(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")
	res, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	e.must(e.db.Exec(`UPDATE payments SET redirect_url = NULL, updated_at = NOW() - INTERVAL '1 minute' WHERE id = ?`, res.PaymentID).Error)

	if _, err := e.hold(u, "A2"); err != nil {
		t.Fatalf("new hold blocked by an orphan checkout: %v", err)
	}
	e.wantStatus(h.BookingID, models.BookingExpired)
	e.checkInvariants()
}

type blockingPublisher struct{}

func (blockingPublisher) Publish(ctx context.Context, _ string, _ []byte) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestConfirm_SlowPublisherDoesNotBlock(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	slow := service.NewBookingService(service.BookingOptions{
		DB:            e.db,
		Repo:          repository.NewBookingRepository(e.db),
		Payments:      repository.NewPaymentRepository(e.db),
		Providers:     e.providers,
		PublicBaseURL: merchantURL,
		HoldTTL:       10 * time.Minute,
		MaxSeats:      4,
		Publisher:     blockingPublisher{},
		Hub:           e.hub,
	})
	h, err := slow.Hold(e.ctx, u, dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("A1")})
	e.must(err)
	pay, err := slow.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	e.capture(pay.TxnRef)

	start := time.Now()
	if ack := slow.HandleNotification(e.ctx, e.mockP, e.notificationRequest(e.mockP, pay.TxnRef)); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("notification took %v with a broker that never confirms", elapsed)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
}

func TestHTTP_LongUserAgentDoesNotBreakHold(t *testing.T) {
	h := newHTTPEnv(t)
	token, _ := h.login(models.RoleCustomer)
	body, err := json.Marshal(map[string]any{"show_id": h.showID, "seat_ids": h.ids("A1")})
	h.must(err)
	req, err := http.NewRequest(http.MethodPost, h.srv.URL+"/api/v1/orders/hold", bytes.NewReader(body))
	h.must(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", strings.Repeat("x", 2000))
	resp, err := http.DefaultClient.Do(req)
	h.must(err)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("hold with a long User-Agent: HTTP %d, want 201", resp.StatusCode)
	}
	if n := h.count(`SELECT COUNT(*) FROM audit_logs WHERE action = 'orders.hold' AND outcome = 'success'
		AND length(user_agent) = 512`); n != 1 {
		t.Fatalf("audit rows with a cut User-Agent = %d, want 1", n)
	}
}
