// Package payment is the provider-agnostic payment contract. Every gateway —
// the demo mock as much as a real one (VNPay, MoMo, ZaloPay...) — is an
// adapter implementing Provider and is registered in the Registry from config.
// The booking service only ever talks to this interface.
//
// Lifecycle of one payment attempt (table payments):
//
//  1. POST /orders/:id/pay       -> the attempt row is stored, then CreatePayment
//     returns the checkout URL the customer is redirected to.
//  2. Gateway -> /payments/{provider}/ipn   (server to server, GET or POST)
//     ParseNotification verifies + decodes it; the booking is confirmed or
//     refunded; AckNotification answers in the gateway's own format.
//  3. Browser -> /payments/{provider}/return
//     ParseReturn verifies the redirect; the service then asks QueryStatus —
//     a return URL is never proof of payment.
//  4. Lost IPN: reads and the sweep job call QueryStatus (reconcile).
//  5. Failed confirm / duplicate payment: Refund, retried by the sweep.
//
// Adding a real gateway: create a package (e.g. internal/payment/vnpay)
// implementing Provider, add its config block under payment.providers and
// register it in cmd/server buildPaymentProviders. Nothing else changes.
package payment

import (
	"errors"
	"net/http"
	"time"

	"context"
)

// State is the provider-reported state of a transaction.
type State string

const (
	StatePending State = "pending"
	StatePaid    State = "paid"
	StateFailed  State = "failed"
)

var (
	// ErrInvalidSignature: the request was not signed by the provider.
	ErrInvalidSignature = errors.New("invalid provider signature")
	// ErrMalformed: the request is not a valid provider message.
	ErrMalformed = errors.New("malformed provider request")
	// ErrGatewayDown: the provider refuses new payments for now.
	ErrGatewayDown = errors.New("payment gateway unavailable")
	// ErrUnknownTxn: the provider does not know this transaction.
	ErrUnknownTxn = errors.New("unknown transaction")
)

// CreateRequest opens a checkout. Amounts are integer VND; an adapter converts
// to its gateway's unit (VNPay expects x100).
type CreateRequest struct {
	TxnRef      string // merchant reference, unique per provider
	BookingID   string
	Amount      int64
	Description string
	ClientIP    string
	Locale      string
	ReturnURL   string // browser comes back here
	NotifyURL   string // server-to-server IPN target (some gateways configure it in their portal instead)
	ExpiresAt   time.Time
}

// Checkout is where to send the customer.
type Checkout struct {
	RedirectURL   string
	ProviderTxnID string            // gateway's own id when known at creation
	Data          map[string]string // anything the adapter needs later (query/refund)
}

// Notification is a verified, normalized gateway message (IPN or query result).
type Notification struct {
	TxnRef        string
	Status        State
	Amount        int64 // amount the gateway actually settled
	ProviderTxnID string
	Data          map[string]string
}

// Transaction is a stored attempt handed back to the adapter for query/refund.
type Transaction struct {
	TxnRef        string
	Amount        int64
	PaidAmount    int64
	ProviderTxnID string
	CreatedAt     time.Time
	PaidAt        *time.Time
	Data          map[string]string
}

// RefundRequest returns collected money.
type RefundRequest struct {
	Amount int64
	Reason string
}

// AckStatus is the merchant verdict on a notification; each adapter maps it to
// the response format and HTTP status its gateway expects.
type AckStatus int

const (
	AckProcessed        AckStatus = iota // applied (booking confirmed, refunded or attempt failed)
	AckDuplicate                         // already applied earlier
	AckUnknownTxn                        // no such payment attempt
	AckInvalidSignature                  // forged or corrupted
	AckInvalid                           // not understood
	AckRetryLater                        // temporary failure, the gateway should retry
)

func (a AckStatus) String() string {
	switch a {
	case AckProcessed:
		return "processed"
	case AckDuplicate:
		return "duplicate"
	case AckUnknownTxn:
		return "unknown transaction"
	case AckInvalidSignature:
		return "invalid signature"
	case AckInvalid:
		return "invalid request"
	default:
		return "retry later"
	}
}

// Provider is one payment gateway adapter.
type Provider interface {
	// Name is the stable id used in URLs, config and payments.provider.
	Name() string
	// DisplayName is shown to customers choosing how to pay.
	DisplayName() string
	CreatePayment(ctx context.Context, req CreateRequest) (*Checkout, error)
	// ParseNotification verifies and decodes an IPN request.
	ParseNotification(r *http.Request) (*Notification, error)
	// AckNotification writes the answer the gateway expects.
	AckNotification(w http.ResponseWriter, ack AckStatus)
	// ParseReturn verifies the browser return redirect and returns TxnRef.
	ParseReturn(r *http.Request) (txnRef string, err error)
	// QueryStatus asks the gateway for the real state of a transaction.
	QueryStatus(ctx context.Context, txn Transaction) (*Notification, error)
	Refund(ctx context.Context, txn Transaction, req RefundRequest) error
}

// Simulator is implemented by providers that ship their own fake gateway
// (the mock): the router mounts SimulatorHandler under SimulatorPath.
type Simulator interface {
	SimulatorPath() string
	SimulatorHandler() http.Handler
}
