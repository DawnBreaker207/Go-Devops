package mock

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
)

// TxnState is the gateway-side state of a fake transaction.
type TxnState string

const (
	TxnPending  TxnState = "pending"
	TxnPaid     TxnState = "paid"
	TxnFailed   TxnState = "failed"
	TxnCanceled TxnState = "canceled"
	TxnRefunded TxnState = "refunded"
)

// Txn is one fake transaction held by the gateway.
type Txn struct {
	Ref          string
	GatewayTxnID string
	BookingID    string
	Description  string
	Amount       int64
	PaidAmount   int64
	State        TxnState
	NotifyURL    string
	ReturnURL    string
	CreatedAt    time.Time
}

// CaptureOptions shapes a simulated payment.
type CaptureOptions struct {
	// AmountDelta makes the gateway settle a different amount (E-P4 drill).
	AmountDelta int64
	// Decline makes the payment fail instead of succeed.
	Decline bool
}

// notification is the IPN body the gateway sends.
type notification struct {
	TxnRef       string `json:"txn_ref"`
	Amount       int64  `json:"amount"`
	Status       string `json:"status"`
	GatewayTxnID string `json:"gateway_txn_id"`
	Signature    string `json:"signature"`
}

// Gateway is the simulated payment service provider: it keeps transactions in
// memory, serves the checkout page, sends signed IPNs over HTTP to the
// merchant notify URL and redirects the browser to the signed return URL —
// the same things a real gateway does from its own servers. A restart forgets
// every transaction.
type Gateway struct {
	secret   string
	baseURL  string
	basePath string
	client   *http.Client

	mu      sync.Mutex
	txns    map[string]*Txn
	down    bool
	refunds int
}

var errNotPayable = errors.New("transaction can no longer be paid")

func newGateway(secret, baseURL, basePath string, client *http.Client) *Gateway {
	return &Gateway{
		secret:   secret,
		baseURL:  strings.TrimRight(baseURL, "/"),
		basePath: basePath,
		client:   client,
		txns:     make(map[string]*Txn),
	}
}

func (g *Gateway) open(req payment.CreateRequest) (Txn, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.down {
		return Txn{}, payment.ErrGatewayDown
	}
	if t, ok := g.txns[req.TxnRef]; ok {
		return *t, nil
	}
	t := &Txn{
		Ref:          req.TxnRef,
		GatewayTxnID: "MOCK" + randomHex(6),
		BookingID:    req.BookingID,
		Description:  req.Description,
		Amount:       req.Amount,
		State:        TxnPending,
		NotifyURL:    req.NotifyURL,
		ReturnURL:    req.ReturnURL,
		CreatedAt:    time.Now(),
	}
	g.txns[t.Ref] = t
	return *t, nil
}

// CheckoutURL is the page the customer is redirected to.
func (g *Gateway) CheckoutURL(ref string) string {
	return g.baseURL + g.basePath + "/checkout?ref=" + url.QueryEscape(ref)
}

// Lookup returns a copy of a transaction.
func (g *Gateway) Lookup(ref string) (Txn, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	t, ok := g.txns[ref]
	if !ok {
		return Txn{}, false
	}
	return *t, true
}

// Capture settles a pending transaction the way the customer's action on the
// checkout page would. Pressing again on a settled transaction changes nothing.
func (g *Gateway) Capture(ref string, opts CaptureOptions) (Txn, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	t, ok := g.txns[ref]
	if !ok {
		return Txn{}, payment.ErrUnknownTxn
	}
	switch t.State {
	case TxnPending:
		if opts.Decline {
			t.State = TxnFailed
		} else {
			t.State = TxnPaid
			t.PaidAmount = t.Amount + opts.AmountDelta
		}
	case TxnPaid, TxnFailed:
	default:
		return *t, errNotPayable
	}
	return *t, nil
}

// Cancel abandons a pending checkout (E-P6): no IPN is sent.
func (g *Gateway) Cancel(ref string) (Txn, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	t, ok := g.txns[ref]
	if !ok {
		return Txn{}, payment.ErrUnknownTxn
	}
	if t.State == TxnPending {
		t.State = TxnCanceled
	}
	return *t, nil
}

// NotificationRequest builds the signed IPN of a settled transaction, exactly
// as Deliver sends it.
func (g *Gateway) NotificationRequest(ctx context.Context, ref string) (*http.Request, error) {
	t, ok := g.Lookup(ref)
	if !ok {
		return nil, payment.ErrUnknownTxn
	}
	status, amount := "", t.PaidAmount
	switch t.State {
	case TxnPaid, TxnRefunded:
		status = string(TxnPaid)
	case TxnFailed:
		status, amount = string(TxnFailed), t.Amount
	default:
		return nil, fmt.Errorf("transaction %s has nothing to notify (%s)", ref, t.State)
	}
	body, err := json.Marshal(notification{
		TxnRef:       ref,
		Amount:       amount,
		Status:       status,
		GatewayTxnID: t.GatewayTxnID,
		Signature:    g.sign(ref, strconv.FormatInt(amount, 10), status, t.GatewayTxnID),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.NotifyURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// Deliver posts the IPN to the merchant, retrying network errors and 5xx
// answers like a real gateway. It returns the last HTTP status.
func (g *Gateway) Deliver(ctx context.Context, ref string) (int, error) {
	status := 0
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return status, ctx.Err()
			case <-time.After(time.Duration(attempt) * 300 * time.Millisecond):
			}
		}
		req, err := g.NotificationRequest(ctx, ref)
		if err != nil {
			return 0, err
		}
		resp, err := g.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		status = resp.StatusCode
		if status < http.StatusInternalServerError {
			return status, nil
		}
		lastErr = fmt.Errorf("merchant answered HTTP %d", status)
	}
	return status, lastErr
}

// ReturnRedirect is the merchant return URL carrying the signed result.
func (g *Gateway) ReturnRedirect(ref string) (string, error) {
	t, ok := g.Lookup(ref)
	if !ok {
		return "", payment.ErrUnknownTxn
	}
	u, err := url.Parse(t.ReturnURL)
	if err != nil {
		return "", fmt.Errorf("bad return url: %w", err)
	}
	q := u.Query()
	q.Set("ref", ref)
	q.Set("status", string(t.State))
	q.Set("sig", g.sign(ref, string(t.State)))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (g *Gateway) refund(ref string, amount int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	t, ok := g.txns[ref]
	if !ok {
		return payment.ErrUnknownTxn
	}
	switch t.State {
	case TxnRefunded:
		return nil
	case TxnPaid:
		if amount != t.PaidAmount {
			return fmt.Errorf("refund amount %d does not match paid amount %d", amount, t.PaidAmount)
		}
		t.State = TxnRefunded
		g.refunds++
		return nil
	default:
		return fmt.Errorf("transaction %s is %s, nothing to refund", ref, t.State)
	}
}

// SetDown makes new payments fail, simulating an outage (E-P10).
func (g *Gateway) SetDown(down bool) {
	g.mu.Lock()
	g.down = down
	g.mu.Unlock()
}

// Refunds counts refunds the gateway accepted.
func (g *Gateway) Refunds() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.refunds
}

func (g *Gateway) sign(parts ...string) string {
	mac := hmac.New(sha256.New, []byte(g.secret))
	mac.Write([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(mac.Sum(nil))
}

func (g *Gateway) verify(signature string, parts ...string) bool {
	return hmac.Equal([]byte(g.sign(parts...)), []byte(signature))
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("read rand: " + err.Error())
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}
