package service_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
)

// Empty uses the default, a disabled provider is refused, the same provider reuses the open checkout.
func TestPay_ProviderChoice(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")

	if _, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "vnpay"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("disabled provider: err = %v, want 400", err)
	}
	def, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{})
	e.must(err)
	if def.Provider != "mock" {
		t.Fatalf("default provider = %s", def.Provider)
	}
	again, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	if again.PaymentID != def.PaymentID {
		t.Fatal("same provider opened a second checkout")
	}
	if n := e.count(`SELECT COUNT(*) FROM payments WHERE booking_id = ?`, h.BookingID); n != 1 {
		t.Fatalf("attempts = %d, want 1", n)
	}
	if _, err := e.svc.Pay(e.ctx, e.users[1], h.BookingID, dto.PayRequest{}); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("someone else's booking: err = %v, want 404", err)
	}
}

// The first settled payment confirms the booking; the other provider's payment is refunded.
func TestTwoProviders_SwitchAndDoublePayment(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	other := e.addProvider("mock2")
	h := e.mustHold(u, "A1", "A2")

	refA := e.pay(u, h.BookingID)
	payB, err := e.svc.Pay(e.ctx, u, h.BookingID, dto.PayRequest{Provider: "mock2"})
	e.must(err)
	if !strings.Contains(payB.RedirectURL, "/mock2-gateway/checkout") {
		t.Fatalf("second provider redirect = %s", payB.RedirectURL)
	}

	e.capture(refA)
	if _, err := other.Gateway().Capture(payB.TxnRef, mock.CaptureOptions{}); err != nil {
		t.Fatal(err)
	}
	if ack := e.ipnVia(other, payB.TxnRef); ack != payment.AckProcessed {
		t.Fatalf("mock2 ack = %s", ack)
	}
	b := e.wantStatus(h.BookingID, models.BookingConfirmed)
	if b.PaymentID == nil || *b.PaymentID != payB.PaymentID {
		t.Fatalf("booking carries payment %v, want %s", b.PaymentID, payB.PaymentID)
	}

	if ack := e.ipn(refA); ack != payment.AckProcessed {
		t.Fatalf("late mock ack = %s", ack)
	}
	pA := e.payment(refA)
	if pA.Status != models.PaymentRefunded || derefStr(pA.StatusReason) != models.PaymentReasonDuplicate {
		t.Fatalf("second payment = %s/%s, want refunded/duplicate", pA.Status, derefStr(pA.StatusReason))
	}
	if txn, _ := e.gw.Lookup(refA); txn.State != mock.TxnRefunded {
		t.Fatalf("mock gateway state = %s", txn.State)
	}
	if other.Gateway().Refunds() != 0 {
		t.Fatal("the payment that bought the tickets was refunded")
	}
	e.checkInvariants()
}

func TestNotify_DeclinedThenRetry(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	h := e.mustHold(u, "A1")
	ref := e.pay(u, h.BookingID)
	e.capture(ref, mock.CaptureOptions{Decline: true})

	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	if p := e.payment(ref); p.Status != models.PaymentFailed || derefStr(p.StatusReason) != models.PaymentReasonDeclined {
		t.Fatalf("payment = %s/%s", p.Status, derefStr(p.StatusReason))
	}
	e.wantStatus(h.BookingID, models.BookingPending)

	ref2 := e.pay(u, h.BookingID)
	if ref2 == ref {
		t.Fatal("retry reused the declined attempt")
	}
	e.capture(ref2)
	if ack := e.ipn(ref2); ack != payment.AckProcessed {
		t.Fatalf("retry ack = %s", ack)
	}
	e.wantStatus(h.BookingID, models.BookingConfirmed)
	e.checkInvariants()
}

func TestReturn_ReconcilesAndRejectsForgery(t *testing.T) {
	e := newEnv(t)
	h := e.mustHold(e.users[0], "A1")
	ref := e.pay(e.users[0], h.BookingID)
	e.capture(ref)
	target, err := e.gw.ReturnRedirect(ref)
	e.must(err)
	if !strings.HasPrefix(target, merchantURL+"/api/v1/payments/mock/return?") {
		t.Fatalf("return target = %s", target)
	}

	forged := strings.Replace(target, "sig=", "sig=00", 1)
	if _, err := e.svc.HandleReturn(e.ctx, e.mockP, httptest.NewRequest(http.MethodGet, forged, nil)); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("forged return: err = %v, want 401", err)
	}
	e.wantStatus(h.BookingID, models.BookingPending)

	res, err := e.svc.HandleReturn(e.ctx, e.mockP, httptest.NewRequest(http.MethodGet, target, nil))
	e.must(err)
	if res.BookingID != h.BookingID || res.BookingStatus != models.BookingConfirmed || res.PaymentStatus != models.PaymentPaid {
		t.Fatalf("return result = %+v", res)
	}
	e.checkInvariants()
}

func TestSweep_ReconcilesAndAbandonsAttempts(t *testing.T) {
	e := newEnv(t)
	paid := e.mustHold(e.users[0], "A1")
	paidRef := e.pay(e.users[0], paid.BookingID)
	e.capture(paidRef)
	idle := e.mustHold(e.users[1], "A2")
	idleRef := e.pay(e.users[1], idle.BookingID)
	e.must(e.db.Exec(`UPDATE payments SET created_at = NOW() - INTERVAL '3 minutes'`).Error)

	res, err := e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.ReconciledPayments != 1 || res.AbandonedPayments != 0 {
		t.Fatalf("first sweep = %+v", res)
	}
	e.wantStatus(paid.BookingID, models.BookingConfirmed)
	if p := e.payment(idleRef); p.Status != models.PaymentPending || p.CheckedAt == nil {
		t.Fatalf("idle attempt = %s checked_at=%v", p.Status, p.CheckedAt)
	}

	e.shiftHold(idle.BookingID, -1200)
	e.must(e.db.Exec(`UPDATE payments SET expires_at = NOW() - INTERVAL '20 minutes', checked_at = NULL WHERE txn_ref = ?`, idleRef).Error)
	res, err = e.svc.SweepExpired(e.ctx, 500)
	e.must(err)
	if res.AbandonedPayments != 1 || res.ExpiredBookings != 1 {
		t.Fatalf("second sweep = %+v", res)
	}
	if p := e.payment(idleRef); p.Status != models.PaymentFailed || derefStr(p.StatusReason) != models.PaymentReasonAbandoned {
		t.Fatalf("idle attempt = %s/%s", p.Status, derefStr(p.StatusReason))
	}
	e.wantStatus(idle.BookingID, models.BookingExpired)
	e.checkInvariants()
}

func TestNotify_UnknownAndMalformedAreAudited(t *testing.T) {
	e := newEnv(t)
	_, err := e.mockP.CreatePayment(e.ctx, payment.CreateRequest{
		TxnRef: "GHOST1", Amount: 1000,
		NotifyURL: merchantURL + "/api/v1/payments/mock/ipn", ReturnURL: merchantURL + "/api/v1/payments/mock/return",
	})
	e.must(err)
	e.capture("GHOST1")
	if ack := e.ipn("GHOST1"); ack != payment.AckUnknownTxn {
		t.Fatalf("unknown txn ack = %s", ack)
	}
	malformed := httptest.NewRequest(http.MethodPost, "/api/v1/payments/mock/ipn", strings.NewReader("not json"))
	if ack := e.notify(e.mockP, malformed); ack != payment.AckInvalid {
		t.Fatalf("malformed ack = %s", ack)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE action = 'payments.notify' AND outcome = 'failure'`); n != 2 {
		t.Fatalf("rejection audit rows = %d, want 2", n)
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
