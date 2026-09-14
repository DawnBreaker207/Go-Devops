package mock

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
)

func openTxn(t *testing.T, p *Provider, ref string, amount int64) {
	t.Helper()
	_, err := p.CreatePayment(context.Background(), payment.CreateRequest{
		TxnRef: ref, Amount: amount, NotifyURL: "http://merchant.test/ipn", ReturnURL: "http://merchant.test/return?x=1",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNotificationSignature(t *testing.T) {
	p := New(Options{Secret: "secret"})
	openTxn(t, p, "REF1", 50000)
	if _, err := p.Gateway().Capture("REF1", CaptureOptions{}); err != nil {
		t.Fatal(err)
	}
	req, err := p.Gateway().NotificationRequest(context.Background(), "REF1")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(req.Body)

	n, err := p.ParseNotification(httptest.NewRequest(http.MethodPost, "/ipn", strings.NewReader(string(body))))
	if err != nil || n.Status != payment.StatePaid || n.Amount != 50000 || n.ProviderTxnID == "" {
		t.Fatalf("parse = %+v, %v", n, err)
	}

	tampered := strings.Replace(string(body), `"amount":50000`, `"amount":1`, 1)
	if _, err := p.ParseNotification(httptest.NewRequest(http.MethodPost, "/ipn", strings.NewReader(tampered))); !errors.Is(err, payment.ErrInvalidSignature) {
		t.Fatalf("tampered amount: err = %v", err)
	}
	other := New(Options{Secret: "other-secret"})
	if _, err := other.ParseNotification(httptest.NewRequest(http.MethodPost, "/ipn", strings.NewReader(string(body)))); !errors.Is(err, payment.ErrInvalidSignature) {
		t.Fatalf("foreign secret: err = %v", err)
	}
	if _, err := p.ParseNotification(httptest.NewRequest(http.MethodPost, "/ipn", strings.NewReader("{"))); !errors.Is(err, payment.ErrMalformed) {
		t.Fatalf("garbage: err = %v", err)
	}
}

func TestReturnSignature(t *testing.T) {
	p := New(Options{Secret: "secret"})
	openTxn(t, p, "REF1", 50000)
	_, _ = p.Gateway().Capture("REF1", CaptureOptions{})
	target, err := p.Gateway().ReturnRedirect("REF1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(target, "x=1") {
		t.Fatalf("merchant query lost: %s", target)
	}
	if ref, err := p.ParseReturn(httptest.NewRequest(http.MethodGet, target, nil)); err != nil || ref != "REF1" {
		t.Fatalf("return = %q, %v", ref, err)
	}
	forged := strings.Replace(target, "status=paid", "status=failed", 1)
	if _, err := p.ParseReturn(httptest.NewRequest(http.MethodGet, forged, nil)); !errors.Is(err, payment.ErrInvalidSignature) {
		t.Fatalf("forged return: err = %v", err)
	}
}

func TestAckStatuses(t *testing.T) {
	p := New(Options{Secret: "s"})
	want := map[payment.AckStatus]int{
		payment.AckProcessed:        200,
		payment.AckDuplicate:        200,
		payment.AckUnknownTxn:       404,
		payment.AckInvalidSignature: 401,
		payment.AckInvalid:          400,
		payment.AckRetryLater:       503,
	}
	for ack, status := range want {
		rec := httptest.NewRecorder()
		p.AckNotification(rec, ack)
		if rec.Code != status {
			t.Errorf("%s: HTTP %d, want %d", ack, rec.Code, status)
		}
	}
}

func TestRefundAndQuery(t *testing.T) {
	ctx := context.Background()
	p := New(Options{Secret: "s"})
	openTxn(t, p, "REF1", 50000)
	txn := payment.Transaction{TxnRef: "REF1"}

	if err := p.Refund(ctx, txn, payment.RefundRequest{Amount: 50000}); err == nil {
		t.Fatal("refund of an unpaid transaction accepted")
	}
	if _, err := p.Gateway().Capture("REF1", CaptureOptions{AmountDelta: 1000}); err != nil {
		t.Fatal(err)
	}
	if n, _ := p.QueryStatus(ctx, txn); n.Status != payment.StatePaid || n.Amount != 51000 {
		t.Fatalf("query = %+v", n)
	}
	if err := p.Refund(ctx, txn, payment.RefundRequest{Amount: 50000}); err == nil {
		t.Fatal("refund of the wrong amount accepted")
	}
	for i := 0; i < 2; i++ {
		if err := p.Refund(ctx, txn, payment.RefundRequest{Amount: 51000}); err != nil {
			t.Fatal(err)
		}
	}
	if p.Gateway().Refunds() != 1 {
		t.Fatalf("refunds = %d", p.Gateway().Refunds())
	}
	if _, err := p.QueryStatus(ctx, payment.Transaction{TxnRef: "NOPE"}); !errors.Is(err, payment.ErrUnknownTxn) {
		t.Fatalf("unknown ref: err = %v", err)
	}
	p.Gateway().SetDown(true)
	if _, err := p.CreatePayment(ctx, payment.CreateRequest{TxnRef: "REF2", Amount: 1}); !errors.Is(err, payment.ErrGatewayDown) {
		t.Fatalf("gateway down: err = %v", err)
	}
}

func TestCheckoutOverHTTP(t *testing.T) {
	var p *Provider
	var mu sync.Mutex
	var received []*payment.Notification

	merchant := http.NewServeMux()
	merchant.HandleFunc("POST /ipn", func(w http.ResponseWriter, r *http.Request) {
		n, err := p.ParseNotification(r)
		if err != nil {
			p.AckNotification(w, payment.AckInvalidSignature)
			return
		}
		mu.Lock()
		received = append(received, n)
		mu.Unlock()
		p.AckNotification(w, payment.AckProcessed)
	})
	merchantSrv := httptest.NewServer(merchant)
	defer merchantSrv.Close()

	p = New(Options{Secret: "s"})
	gwSrv := httptest.NewServer(p.SimulatorHandler())
	defer gwSrv.Close()

	for _, tc := range []struct {
		ref, mode string
		ipn       payment.State
	}{
		{"PAY1", "pay", payment.StatePaid},
		{"DECLINE1", "decline", payment.StateFailed},
		{"NOIPN1", "pay_no_ipn", ""},
		{"CANCEL1", "cancel", ""},
	} {
		_, err := p.CreatePayment(context.Background(), payment.CreateRequest{
			TxnRef: tc.ref, Amount: 50000, Description: "Ve xem phim",
			NotifyURL: merchantSrv.URL + "/ipn", ReturnURL: merchantSrv.URL + "/return",
		})
		if err != nil {
			t.Fatal(err)
		}
		page, err := http.Get(gwSrv.URL + p.SimulatorPath() + "/checkout?ref=" + tc.ref)
		if err != nil {
			t.Fatal(err)
		}
		html, _ := io.ReadAll(page.Body)
		page.Body.Close()
		if page.StatusCode != 200 || !strings.Contains(string(html), "50.000 ₫") {
			t.Fatalf("%s: checkout page HTTP %d", tc.ref, page.StatusCode)
		}

		mu.Lock()
		before := len(received)
		mu.Unlock()
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := client.PostForm(gwSrv.URL+p.SimulatorPath()+"/checkout/"+tc.ref, url.Values{"mode": {tc.mode}})
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		loc := resp.Header.Get("Location")
		if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, merchantSrv.URL+"/return?") {
			t.Fatalf("%s: HTTP %d location %q", tc.ref, resp.StatusCode, loc)
		}
		if ref, err := p.ParseReturn(httptest.NewRequest(http.MethodGet, loc, nil)); err != nil || ref != tc.ref {
			t.Fatalf("%s: return verify %q %v", tc.ref, ref, err)
		}

		mu.Lock()
		got := received[before:]
		mu.Unlock()
		switch {
		case tc.ipn == "" && len(got) != 0:
			t.Fatalf("%s: unexpected IPN %+v", tc.ref, got)
		case tc.ipn != "" && (len(got) != 1 || got[0].Status != tc.ipn || got[0].TxnRef != tc.ref):
			t.Fatalf("%s: IPNs %+v, want one %s", tc.ref, got, tc.ipn)
		}
	}
}
