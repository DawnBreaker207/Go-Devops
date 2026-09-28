package vnpay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
)

// Amounts are whole VND elsewhere; the x100 conversion lives here only.
const amountMultiplier = 100

// All VNPay timestamps are GMT+7; the adapter carries its own location.
const dateLayout = "20060102150405"

const (
	apiVersion    = "2.1.0"
	commandPay    = "pay"
	commandQuery  = "querydr"
	commandRefund = "refund"
	currencyVND   = "VND"
	defaultOrderType = "other"
)

// Both vnp_ResponseCode AND vnp_TransactionStatus must be "00" before money counts as collected.
const responseCodeSuccess = "00"

type Options struct {
	DisplayName string
	TmnCode string
	HashSecret string
	PayURL string
	APIURL string
	Locale string
	HTTPClient *http.Client
	now func() time.Time
}

type Provider struct {
	display string
	tmnCode string
	secret  string
	payURL  string
	apiURL  string
	locale  string
	client  *http.Client
	loc     *time.Location
	now     func() time.Time
}

// Compile-time proof of the contract. Deliberately NOT payment.Simulator: the simulator
// interface is what mounts a fake gateway, and a real provider must never have one.
var _ payment.Provider = (*Provider)(nil)

func New(opts Options) (*Provider, error) {
	if strings.TrimSpace(opts.TmnCode) == "" {
		return nil, fmt.Errorf("vnpay: tmn_code is required")
	}
	if strings.TrimSpace(opts.HashSecret) == "" {
		return nil, fmt.Errorf("vnpay: hash_secret is required")
	}
	if strings.TrimSpace(opts.PayURL) == "" {
		return nil, fmt.Errorf("vnpay: pay_url is required")
	}
	if opts.DisplayName == "" {
		opts.DisplayName = "VNPay"
	}
	if opts.Locale == "" {
		opts.Locale = "vn"
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if opts.now == nil {
		opts.now = time.Now
	}
	// VNPay timestamps are GMT+7. FixedZone rather than LoadLocation so the adapter does
	// not depend on a tzdata file being present in the container.
	loc := time.FixedZone("ICT", 7*60*60)
	return &Provider{
		display: opts.DisplayName,
		tmnCode: opts.TmnCode,
		secret:  opts.HashSecret,
		payURL:  opts.PayURL,
		apiURL:  opts.APIURL,
		locale:  opts.Locale,
		client:  opts.HTTPClient,
		loc:     loc,
		now:     opts.now,
	}, nil
}

func (p *Provider) Name() string        { return "vnpay" }
func (p *Provider) DisplayName() string { return p.display }

func (p *Provider) CreatePayment(_ context.Context, req payment.CreateRequest) (*payment.Checkout, error) {
	now := p.now().In(p.loc)

	params := map[string]string{
		"vnp_Version":    apiVersion,
		"vnp_Command":    commandPay,
		"vnp_TmnCode":    p.tmnCode,
		"vnp_Amount":     strconv.FormatInt(req.Amount*amountMultiplier, 10),
		"vnp_CreateDate": now.Format(dateLayout),
		"vnp_CurrCode":   currencyVND,
		"vnp_IpAddr":     ipOrFallback(req.ClientIP),
		"vnp_Locale":     localeOrDefault(req.Locale, p.locale),
		"vnp_OrderInfo":  orderInfo(req),
		"vnp_OrderType":  defaultOrderType,
		"vnp_ReturnUrl":  req.ReturnURL,
		"vnp_TxnRef":     req.TxnRef,
	}
	if !req.ExpiresAt.IsZero() {
		params["vnp_ExpireDate"] = req.ExpiresAt.In(p.loc).Format(dateLayout)
	}

	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	q.Set(fieldSecureHash, signData(p.secret, params))

	sep := "?"
	if strings.Contains(p.payURL, "?") {
		sep = "&"
	}
	return &payment.Checkout{RedirectURL: p.payURL + sep + q.Encode()}, nil
}

// ParseNotification verifies the signature BEFORE reading anything else out of the
// request, then decodes VNPay's two status fields.
//
// Money is only collected when vnp_ResponseCode AND vnp_TransactionStatus are both "00".
// Anything else is a genuine failure; there is no pending state on this callback.
func (p *Provider) ParseNotification(r *http.Request) (*payment.Notification, error) {
	q, err := callbackValues(r)
	if err != nil {
		return nil, err
	}
	params := queryParams(q)

	ref := params["vnp_TxnRef"]
	if ref == "" {
		return nil, fmt.Errorf("%w: missing vnp_TxnRef", payment.ErrMalformed)
	}
	if !verify(p.secret, params, params[fieldSecureHash]) {
		return nil, payment.ErrInvalidSignature
	}

	amount, err := parseAmount(params["vnp_Amount"])
	if err != nil {
		return nil, err
	}

	state := payment.StateFailed
	if params["vnp_ResponseCode"] == responseCodeSuccess && params["vnp_TransactionStatus"] == responseCodeSuccess {
		state = payment.StatePaid
	}
	return &payment.Notification{
		TxnRef:        ref,
		Status:        state,
		Amount:        amount,
		ProviderTxnID: params["vnp_TransactionNo"],
		Data: map[string]string{
			"vnp_ResponseCode":      params["vnp_ResponseCode"],
			"vnp_TransactionStatus": params["vnp_TransactionStatus"],
			"vnp_BankCode":          params["vnp_BankCode"],
			"vnp_PayDate":           params["vnp_PayDate"],
		},
	}, nil
}

// AckNotification answers the IPN the way VNPay reads it.
//
// Two rules that are easy to get wrong and expensive when you do:
//
//   - the HTTP status is ALWAYS 200. VNPay reads the JSON body, not the status line, so
//     a 4xx here reads as a transport failure rather than the verdict it carries.
//   - "00" and "02" end the conversation; "01", "04", "97" and "99" make VNPay retry, up
//     to 10 times at 5-minute intervals. Mapping a permanent refusal onto a retryable
//     code therefore buys an hour of pointless callbacks, and mapping a temporary failure
//     onto "00" throws away the retry that would have fixed it.
func (p *Provider) AckNotification(w http.ResponseWriter, ack payment.AckStatus) {
	code, message := "00", "Confirm Success"
	switch ack {
	case payment.AckProcessed:
	case payment.AckDuplicate:
		code, message = "02", "Order already confirmed"
	case payment.AckUnknownTxn:
		code, message = "01", "Order not found"
	case payment.AckInvalidSignature:
		code, message = "97", "Invalid Checksum"
	case payment.AckInvalid:
		code, message = "99", "Invalid request"
	default: // AckRetryLater
		code, message = "99", "Unknown error, please retry"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"RspCode": code, "Message": message})
}

// ParseReturn verifies the browser redirect and returns the reference only.
//
// It deliberately reports nothing about the outcome: a return is a browser being sent
// back, not a settlement. The caller re-reads the transaction through QueryStatus.
func (p *Provider) ParseReturn(r *http.Request) (string, error) {
	params := queryParams(r.URL.Query())
	ref := params["vnp_TxnRef"]
	if ref == "" {
		return "", fmt.Errorf("%w: missing vnp_TxnRef", payment.ErrMalformed)
	}
	if !verify(p.secret, params, params[fieldSecureHash]) {
		return "", payment.ErrInvalidSignature
	}
	return ref, nil
}

// The merchant commands do NOT sign a sorted query string. Their checksum is a
// PIPE-JOINED concatenation of named fields in a fixed order, and the order differs
// between querydr and refund. Reusing the checkout signer here silently produces a
// wrong hash, which VNPay answers as a checksum failure rather than a helpful error.

func (p *Provider) QueryStatus(ctx context.Context, txn payment.Transaction) (*payment.Notification, error) {
	if p.apiURL == "" {
		return nil, fmt.Errorf("%w: vnpay api_url is not configured", payment.ErrGatewayDown)
	}
	now := p.now().In(p.loc)
	requestID := newRequestID()
	createDate := now.Format(dateLayout)
	txnDate := txn.CreatedAt.In(p.loc).Format(dateLayout)
	orderInfo := "Query " + txn.TxnRef

	body := map[string]string{
		"vnp_RequestId":       requestID,
		"vnp_Version":         apiVersion,
		"vnp_Command":         commandQuery,
		"vnp_TmnCode":         p.tmnCode,
		"vnp_TxnRef":          txn.TxnRef,
		"vnp_OrderInfo":       orderInfo,
		"vnp_TransactionDate": txnDate,
		"vnp_CreateDate":      createDate,
		"vnp_IpAddr":          ipOrFallback(""),
	}
	body[fieldSecureHash] = pipeHash(p.secret,
		requestID, apiVersion, commandQuery, p.tmnCode, txn.TxnRef,
		txnDate, createDate, body["vnp_IpAddr"], orderInfo)

	res, err := p.postCommand(ctx, body)
	if err != nil {
		return nil, err
	}

	amount, err := parseAmount(res["vnp_Amount"])
	if err != nil {
		// A querydr for a transaction VNPay has never seen carries no amount; that is
		// not a malformed reply, it is an answer.
		amount = 0
	}
	state := payment.StatePending
	switch {
	case res["vnp_ResponseCode"] == responseCodeSuccess && res["vnp_TransactionStatus"] == responseCodeSuccess:
		state = payment.StatePaid
	case res["vnp_TransactionStatus"] == "01": // still being processed at the bank
		state = payment.StatePending
	case res["vnp_ResponseCode"] == "91": // no matching transaction found
		return nil, payment.ErrUnknownTxn
	default:
		state = payment.StateFailed
	}
	return &payment.Notification{
		TxnRef:        txn.TxnRef,
		Status:        state,
		Amount:        amount,
		ProviderTxnID: res["vnp_TransactionNo"],
		Data:          res,
	}, nil
}

func (p *Provider) Refund(ctx context.Context, txn payment.Transaction, req payment.RefundRequest) error {
	if p.apiURL == "" {
		return fmt.Errorf("%w: vnpay api_url is not configured", payment.ErrGatewayDown)
	}
	now := p.now().In(p.loc)
	requestID := newRequestID()
	createDate := now.Format(dateLayout)
	txnDate := txn.CreatedAt.In(p.loc).Format(dateLayout)
	amount := strconv.FormatInt(req.Amount*amountMultiplier, 10)
	// "02" is a full refund, "03" a partial one. Sending "02" for a partial amount is
	// accepted and refunds the WHOLE transaction, so this must follow the real figures.
	txnType := "02"
	if txn.PaidAmount > 0 && req.Amount < txn.PaidAmount {
		txnType = "03"
	}
	createBy := "system"
	orderInfo := refundReason(req.Reason, txn.TxnRef)
	ip := ipOrFallback("")

	body := map[string]string{
		"vnp_RequestId":       requestID,
		"vnp_Version":         apiVersion,
		"vnp_Command":         commandRefund,
		"vnp_TmnCode":         p.tmnCode,
		"vnp_TransactionType": txnType,
		"vnp_TxnRef":          txn.TxnRef,
		"vnp_Amount":          amount,
		"vnp_TransactionNo":   txn.ProviderTxnID,
		"vnp_TransactionDate": txnDate,
		"vnp_CreateBy":        createBy,
		"vnp_CreateDate":      createDate,
		"vnp_IpAddr":          ip,
		"vnp_OrderInfo":       orderInfo,
	}
	body[fieldSecureHash] = pipeHash(p.secret,
		requestID, apiVersion, commandRefund, p.tmnCode, txnType, txn.TxnRef,
		amount, txn.ProviderTxnID, txnDate, createBy, createDate, ip, orderInfo)

	res, err := p.postCommand(ctx, body)
	if err != nil {
		return err
	}
	if res["vnp_ResponseCode"] != responseCodeSuccess {
		return fmt.Errorf("vnpay refund refused: %s %s", res["vnp_ResponseCode"], res["vnp_Message"])
	}
	return nil
}

func (p *Provider) postCommand(ctx context.Context, body map[string]string) (map[string]string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrGatewayDown, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrGatewayDown, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", payment.ErrGatewayDown, res.StatusCode)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrMalformed, err)
	}
	out := make(map[string]string, len(decoded))
	for k, v := range decoded {
		switch t := v.(type) {
		case string:
			out[k] = t
		case float64:
			out[k] = strconv.FormatInt(int64(t), 10)
		}
	}
	return out, nil
}

// pipeHash is the merchant-API checksum: the named fields joined with "|", in the order
// the caller passes them, HMAC-SHA512'd with the merchant secret. No sorting, no URL
// encoding - it shares nothing with the checkout signer but the algorithm.
func pipeHash(secret string, fields ...string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(strings.Join(fields, "|")))
	return hex.EncodeToString(mac.Sum(nil))
}

// callbackValues reads the parameters from wherever this callback carries them. VNPay
// sends the IPN as a GET query, but the route also accepts POST, and a form-encoded body
// must not be mistaken for an empty callback.
func callbackValues(r *http.Request) (url.Values, error) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("%w: %v", payment.ErrMalformed, err)
		}
		if len(r.PostForm) > 0 {
			return r.PostForm, nil
		}
	}
	q := r.URL.Query()
	if len(q) == 0 {
		return nil, fmt.Errorf("%w: no parameters", payment.ErrMalformed)
	}
	return q, nil
}

// parseAmount converts VNPay's x100 figure back to whole VND.
func parseAmount(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("%w: missing vnp_Amount", payment.ErrMalformed)
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: vnp_Amount %q", payment.ErrMalformed, raw)
	}
	return v / amountMultiplier, nil
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A collision only costs one rejected request, so a clock fallback is enough.
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

// orderInfo is what the customer reads on VNPay's page. The docs ask for unaccented
// Vietnamese with no special characters, so it is built rather than passed through.
func orderInfo(req payment.CreateRequest) string {
	if s := strings.TrimSpace(req.Description); s != "" {
		return sanitiseOrderInfo(s)
	}
	return "Thanh toan don hang " + sanitiseOrderInfo(req.TxnRef)
}

func refundReason(reason, ref string) string {
	if s := strings.TrimSpace(reason); s != "" {
		return sanitiseOrderInfo(s)
	}
	return "Hoan tien don hang " + sanitiseOrderInfo(ref)
}

// sanitiseOrderInfo keeps ASCII letters, digits and spaces. VNPay rejects the rest, and a
// rejected field surfaces as a checksum error rather than a field error.
func sanitiseOrderInfo(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if len(out) > 255 {
		out = out[:255]
	}
	return out
}

func ipOrFallback(ip string) string {
	if s := strings.TrimSpace(ip); s != "" {
		return s
	}
	return "127.0.0.1"
}

func localeOrDefault(reqLocale, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(reqLocale)) {
	case "vn", "vi":
		return "vn"
	case "en":
		return "en"
	}
	return fallback
}
