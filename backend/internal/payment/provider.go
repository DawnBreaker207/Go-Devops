// Package payment defines the gateway-agnostic Provider contract; each gateway is an adapter registered from config.
package payment

import (
	"errors"
	"net/http"
	"time"

	"context"
)

type State string

const (
	StatePending State = "pending"
	StatePaid    State = "paid"
	StateFailed  State = "failed"
)

var (
	ErrInvalidSignature = errors.New("invalid provider signature")
	ErrMalformed        = errors.New("malformed provider request")
	ErrGatewayDown      = errors.New("payment gateway unavailable")
	ErrUnknownTxn       = errors.New("unknown transaction")
)

// CreateRequest amounts are integer VND; adapters convert to their gateway's unit (VNPay expects x100).
type CreateRequest struct {
	TxnRef      string // unique per provider
	BookingID   string
	Amount      int64
	Description string
	ClientIP    string
	Locale      string
	ReturnURL   string
	NotifyURL   string // some gateways configure the IPN target in their portal instead
	ExpiresAt   time.Time
}

type Checkout struct {
	RedirectURL   string
	ProviderTxnID string            // when known at creation
	Data          map[string]string // anything the adapter needs later for query/refund
}

type Notification struct {
	TxnRef        string
	Status        State
	Amount        int64 // amount the gateway actually settled
	ProviderTxnID string
	Data          map[string]string
}

type Transaction struct {
	TxnRef        string
	Amount        int64
	PaidAmount    int64
	ProviderTxnID string
	CreatedAt     time.Time
	PaidAt        *time.Time
	Data          map[string]string
}

type RefundRequest struct {
	Amount int64
	Reason string
}

// AckStatus is the verdict on a notification; each adapter maps it to the response its gateway expects.
type AckStatus int

const (
	AckProcessed AckStatus = iota // booking confirmed, refunded or attempt failed
	AckDuplicate
	AckUnknownTxn
	AckInvalidSignature
	AckInvalid
	AckRetryLater // temporary failure, the gateway should retry
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

type Provider interface {
	// Name is the stable id used in URLs, config and payments.provider.
	Name() string
	DisplayName() string
	CreatePayment(ctx context.Context, req CreateRequest) (*Checkout, error)
	// ParseNotification must verify the provider signature before decoding.
	ParseNotification(r *http.Request) (*Notification, error)
	AckNotification(w http.ResponseWriter, ack AckStatus)
	// ParseReturn verifies the return redirect. A return is never proof of payment: use QueryStatus.
	ParseReturn(r *http.Request) (txnRef string, err error)
	QueryStatus(ctx context.Context, txn Transaction) (*Notification, error)
	Refund(ctx context.Context, txn Transaction, req RefundRequest) error
}

// Simulator providers ship a fake gateway; the router mounts SimulatorHandler under SimulatorPath.
type Simulator interface {
	SimulatorPath() string
	SimulatorHandler() http.Handler
}
