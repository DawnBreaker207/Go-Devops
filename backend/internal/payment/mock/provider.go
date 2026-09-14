// Package mock is a demo payment provider backed by an in-process simulated gateway.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
)

type Options struct {
	// Name defaults to "mock".
	Name        string
	DisplayName string
	// Secret signs IPNs and return redirects (HMAC-SHA256).
	Secret string
	// PublicBaseURL is where browsers reach the checkout page.
	PublicBaseURL string
	// HTTPClient delivers IPNs; defaults to a client with a 5s timeout.
	HTTPClient *http.Client
}

type Provider struct {
	name    string
	display string
	gw      *Gateway
}

var _ payment.Provider = (*Provider)(nil)
var _ payment.Simulator = (*Provider)(nil)

func New(opts Options) *Provider {
	if opts.Name == "" {
		opts.Name = "mock"
	}
	if opts.DisplayName == "" {
		opts.DisplayName = "Cổng thử nghiệm (" + opts.Name + ")"
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &Provider{
		name:    opts.Name,
		display: opts.DisplayName,
		gw:      newGateway(opts.Secret, opts.PublicBaseURL, "/"+opts.Name+"-gateway", opts.HTTPClient),
	}
}

func (p *Provider) Gateway() *Gateway { return p.gw }

func (p *Provider) Name() string        { return p.name }
func (p *Provider) DisplayName() string { return p.display }

func (p *Provider) CreatePayment(_ context.Context, req payment.CreateRequest) (*payment.Checkout, error) {
	t, err := p.gw.open(req)
	if err != nil {
		return nil, err
	}
	return &payment.Checkout{RedirectURL: p.gw.CheckoutURL(t.Ref), ProviderTxnID: t.GatewayTxnID}, nil
}

func (p *Provider) ParseNotification(r *http.Request) (*payment.Notification, error) {
	var n notification
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&n); err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrMalformed, err)
	}
	if n.TxnRef == "" || n.Signature == "" {
		return nil, fmt.Errorf("%w: missing fields", payment.ErrMalformed)
	}
	if !p.gw.verify(n.Signature, n.TxnRef, strconv.FormatInt(n.Amount, 10), n.Status, n.GatewayTxnID) {
		return nil, payment.ErrInvalidSignature
	}
	var state payment.State
	switch TxnState(n.Status) {
	case TxnPaid:
		state = payment.StatePaid
	case TxnFailed:
		state = payment.StateFailed
	default:
		return nil, fmt.Errorf("%w: status %q", payment.ErrMalformed, n.Status)
	}
	return &payment.Notification{TxnRef: n.TxnRef, Status: state, Amount: n.Amount, ProviderTxnID: n.GatewayTxnID}, nil
}

// AckNotification answers with an HTTP status and a VNPay-like code so the gateway
// retries only temporary failures.
func (p *Provider) AckNotification(w http.ResponseWriter, ack payment.AckStatus) {
	status, code := http.StatusOK, "00"
	switch ack {
	case payment.AckProcessed:
	case payment.AckDuplicate:
		code = "02"
	case payment.AckUnknownTxn:
		status, code = http.StatusNotFound, "01"
	case payment.AckInvalidSignature:
		status, code = http.StatusUnauthorized, "97"
	case payment.AckInvalid:
		status, code = http.StatusBadRequest, "99"
	default:
		status, code = http.StatusServiceUnavailable, "98"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": ack.String()})
}

func (p *Provider) ParseReturn(r *http.Request) (string, error) {
	q := r.URL.Query()
	ref, status, sig := q.Get("ref"), q.Get("status"), q.Get("sig")
	if ref == "" || status == "" || sig == "" {
		return "", fmt.Errorf("%w: missing return parameters", payment.ErrMalformed)
	}
	if !p.gw.verify(sig, ref, status) {
		return "", payment.ErrInvalidSignature
	}
	return ref, nil
}

func (p *Provider) QueryStatus(_ context.Context, txn payment.Transaction) (*payment.Notification, error) {
	t, ok := p.gw.Lookup(txn.TxnRef)
	if !ok {
		return nil, payment.ErrUnknownTxn
	}
	n := &payment.Notification{TxnRef: t.Ref, Status: payment.StatePending, Amount: t.Amount, ProviderTxnID: t.GatewayTxnID}
	switch t.State {
	case TxnPaid, TxnRefunded:
		n.Status, n.Amount = payment.StatePaid, t.PaidAmount
	case TxnFailed, TxnCanceled:
		n.Status = payment.StateFailed
	}
	return n, nil
}

func (p *Provider) Refund(ctx context.Context, txn payment.Transaction, req payment.RefundRequest) error {
	return p.gw.refund(ctx, txn.TxnRef, req.Amount)
}

func (p *Provider) SimulatorPath() string          { return p.gw.basePath }
func (p *Provider) SimulatorHandler() http.Handler { return p.gw.Handler() }
