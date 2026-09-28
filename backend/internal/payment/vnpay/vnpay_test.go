package vnpay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
)

const testSecret = "SECRETKEY123456"

func newTestProvider(t *testing.T, apiURL string) *Provider {
	t.Helper()
	fixed := time.Date(2026, 9, 26, 19, 30, 0, 0, time.UTC)
	p, err := New(Options{
		TmnCode:    "CINEMA01",
		HashSecret: testSecret,
		PayURL:     "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html",
		APIURL:     apiURL,
		now:        func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

/* -------------------------------------------------------------------------- */
/* The signing rules themselves                                                */
/* -------------------------------------------------------------------------- */

// The four rules VNPay states, asserted on the STRING that gets hashed rather than on a
// hash compared to another hash - that comparison passes just as happily with both sides
// wrong, which is exactly how a signing bug survives its own test suite.
func TestSignSource_FollowsVNPayRules(t *testing.T) {
	params := map[string]string{
		"vnp_Version":         "2.1.0",
		"vnp_Command":         "pay",
		"vnp_Amount":          "14000000",
		"vnp_OrderInfo":       "Thanh toan don hang 42", // spaces: PHP urlencode -> '+'
		"vnp_ReturnUrl":       "http://localhost:8080/api/v1/payments/vnpay/return",
		"vnp_BankCode":        "", // empty: omitted entirely, never signed as "key="
		fieldSecureHash:       "should-be-ignored",
		fieldSecureHashType:   "SHA512",
		"vnp_TransactionDate": "20260926193000",
	}

	got := signSource(params)

	want := strings.Join([]string{
		"vnp_Amount=14000000",
		"vnp_Command=pay",
		"vnp_OrderInfo=Thanh+toan+don+hang+42",
		"vnp_ReturnUrl=http%3A%2F%2Flocalhost%3A8080%2Fapi%2Fv1%2Fpayments%2Fvnpay%2Freturn",
		"vnp_TransactionDate=20260926193000",
		"vnp_Version=2.1.0",
	}, "&")

	if got != want {
		t.Fatalf("sign source mismatch\n got: %s\nwant: %s", got, want)
	}
}

// Sorting must be by NAME, so the map's iteration order - which Go randomises on purpose -
// can never reach the hash.
func TestSignData_IsOrderIndependent(t *testing.T) {
	a := map[string]string{"vnp_TxnRef": "R1", "vnp_Amount": "100", "vnp_Command": "pay"}
	first := signData(testSecret, a)
	for i := 0; i < 50; i++ {
		if got := signData(testSecret, a); got != first {
			t.Fatalf("signature changed between runs: %s vs %s", got, first)
		}
	}
}

func TestVerify_RejectsTamperingAndAcceptsEitherHexCase(t *testing.T) {
	params := map[string]string{"vnp_TxnRef": "R1", "vnp_Amount": "14000000"}
	sig := signData(testSecret, params)

	if !verify(testSecret, params, sig) {
		t.Fatal("a freshly made signature must verify")
	}
	if !verify(testSecret, params, strings.ToUpper(sig)) {
		t.Fatal("uppercase hex must verify: VNPay SDKs differ on casing")
	}
	if verify(testSecret, params, "") {
		t.Fatal("an empty signature must never verify")
	}
	if verify("other-secret", params, sig) {
		t.Fatal("a different secret must not verify")
	}

	tampered := map[string]string{"vnp_TxnRef": "R1", "vnp_Amount": "1400000"} // one zero less
	if verify(testSecret, tampered, sig) {
		t.Fatal("changing the amount must invalidate the signature")
	}
}

// The merchant API signs a pipe-joined field list, NOT a sorted query string. Reusing the
// checkout signer there produces a hash VNPay rejects as a checksum failure, so the two
// must be provably different functions.
func TestPipeHash_IsNotTheQueryStringSigner(t *testing.T) {
	params := map[string]string{"vnp_Command": "querydr", "vnp_TxnRef": "R1"}
	if pipeHash(testSecret, "querydr", "R1") == signData(testSecret, params) {
		t.Fatal("merchant-API hash must not equal the checkout hash for the same fields")
	}
	if pipeHash(testSecret, "a", "b") == pipeHash(testSecret, "b", "a") {
		t.Fatal("pipe hash must depend on field ORDER - the docs fix a different order per command")
	}
}

/* -------------------------------------------------------------------------- */
/* Checkout                                                                    */
/* -------------------------------------------------------------------------- */

func TestCreatePayment_BuildsAVerifiableURLWithTheAmountTimes100(t *testing.T) {
	p := newTestProvider(t, "")

	out, err := p.CreatePayment(context.Background(), payment.CreateRequest{
		TxnRef:    "CP260926TEST01",
		Amount:    140000, // whole VND
		ClientIP:  "203.0.113.9",
		ReturnURL: "http://localhost:8080/api/v1/payments/vnpay/return",
		ExpiresAt: time.Date(2026, 9, 26, 19, 40, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}

	u, err := url.Parse(out.RedirectURL)
	if err != nil {
		t.Fatalf("redirect url does not parse: %v", err)
	}
	q := u.Query()

	if got := q.Get("vnp_Amount"); got != "14000000" {
		t.Fatalf("vnp_Amount = %s, want 14000000 (140000 VND x100)", got)
	}
	if got := q.Get("vnp_TmnCode"); got != "CINEMA01" {
		t.Fatalf("vnp_TmnCode = %s", got)
	}
	if got := q.Get("vnp_Version"); got != "2.1.0" {
		t.Fatalf("vnp_Version = %s, want 2.1.0", got)
	}
	// GMT+7: 19:30 UTC is 02:30 the next day in cinema time.
	if got := q.Get("vnp_CreateDate"); got != "20260927023000" {
		t.Fatalf("vnp_CreateDate = %s, want it in GMT+7 regardless of server zone", got)
	}
	// The signature we just produced must verify against the parameters as a callback
	// would present them - if it does not, sign and verify have drifted apart.
	if !verify(testSecret, queryParams(q), q.Get(fieldSecureHash)) {
		t.Fatal("the checkout URL does not verify against its own signature")
	}
}

/* -------------------------------------------------------------------------- */
/* Callbacks                                                                   */
/* -------------------------------------------------------------------------- */

func signedCallback(t *testing.T, extra map[string]string) string {
	t.Helper()
	params := map[string]string{
		"vnp_TmnCode":           "CINEMA01",
		"vnp_Amount":            "14000000",
		"vnp_TxnRef":            "CP260926TEST01",
		"vnp_ResponseCode":      "00",
		"vnp_TransactionStatus": "00",
		"vnp_TransactionNo":     "14200001",
	}
	for k, v := range extra {
		params[k] = v
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	q.Set(fieldSecureHash, signData(testSecret, params))
	return "/api/v1/payments/vnpay/ipn?" + q.Encode()
}

func TestParseNotification_PaidOnlyWhenBothStatusFieldsAreZeroZero(t *testing.T) {
	p := newTestProvider(t, "")

	cases := []struct {
		name  string
		extra map[string]string
		want  payment.State
	}{
		{"both 00", nil, payment.StatePaid},
		{"response code not 00", map[string]string{"vnp_ResponseCode": "24"}, payment.StateFailed},
		{"transaction status not 00", map[string]string{"vnp_TransactionStatus": "02"}, payment.StateFailed},
		{"both wrong", map[string]string{"vnp_ResponseCode": "24", "vnp_TransactionStatus": "02"}, payment.StateFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, signedCallback(t, tc.extra), nil)
			n, err := p.ParseNotification(r)
			if err != nil {
				t.Fatalf("ParseNotification: %v", err)
			}
			if n.Status != tc.want {
				t.Fatalf("status = %s, want %s", n.Status, tc.want)
			}
			if n.Amount != 140000 {
				t.Fatalf("amount = %d, want 140000 whole VND (from 14000000)", n.Amount)
			}
		})
	}
}

func TestParseNotification_RejectsForgery(t *testing.T) {
	p := newTestProvider(t, "")

	t.Run("tampered amount", func(t *testing.T) {
		// Sign an honest 140.000 callback, then rewrite the amount in the URL.
		raw := signedCallback(t, nil)
		forged := strings.Replace(raw, "vnp_Amount=14000000", "vnp_Amount=100", 1)
		r := httptest.NewRequest(http.MethodGet, forged, nil)
		if _, err := p.ParseNotification(r); err != payment.ErrInvalidSignature {
			t.Fatalf("err = %v, want ErrInvalidSignature", err)
		}
	})

	t.Run("no signature at all", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet,
			"/ipn?vnp_TxnRef=CP1&vnp_Amount=100&vnp_ResponseCode=00&vnp_TransactionStatus=00", nil)
		if _, err := p.ParseNotification(r); err != payment.ErrInvalidSignature {
			t.Fatalf("err = %v, want ErrInvalidSignature", err)
		}
	})

	t.Run("no parameters", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/ipn", nil)
		if _, err := p.ParseNotification(r); err == nil {
			t.Fatal("an empty callback must not parse")
		}
	})
}

// A POST IPN carrying form fields must be read from the body, not mistaken for empty.
func TestParseNotification_AcceptsAFormEncodedPost(t *testing.T) {
	p := newTestProvider(t, "")
	raw := signedCallback(t, nil)
	body := strings.SplitN(raw, "?", 2)[1]

	r := httptest.NewRequest(http.MethodPost, "/api/v1/payments/vnpay/ipn", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	n, err := p.ParseNotification(r)
	if err != nil {
		t.Fatalf("ParseNotification(POST): %v", err)
	}
	if n.Status != payment.StatePaid {
		t.Fatalf("status = %s, want paid", n.Status)
	}
}

func TestParseReturn_VerifiesAndReportsNothingAboutTheOutcome(t *testing.T) {
	p := newTestProvider(t, "")

	// A return claiming failure still parses: the reference is all it may be trusted for.
	raw := strings.Replace(signedCallback(t, map[string]string{"vnp_ResponseCode": "24"}),
		"/ipn?", "/return?", 1)
	r := httptest.NewRequest(http.MethodGet, raw, nil)
	ref, err := p.ParseReturn(r)
	if err != nil {
		t.Fatalf("ParseReturn: %v", err)
	}
	if ref != "CP260926TEST01" {
		t.Fatalf("ref = %q", ref)
	}

	forged := strings.Replace(raw, "vnp_TxnRef=CP260926TEST01", "vnp_TxnRef=SOMEONE_ELSE", 1)
	if _, err := p.ParseReturn(httptest.NewRequest(http.MethodGet, forged, nil)); err != payment.ErrInvalidSignature {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}
}

// VNPay reads the JSON body, never the status line, and only "00"/"02" end the retries.
// Answering a permanent refusal with a retryable code buys an hour of callbacks; the
// reverse throws away the retry that would have fixed a transient failure.
func TestAckNotification_AlwaysHTTP200WithTheRightRspCode(t *testing.T) {
	p := newTestProvider(t, "")

	cases := map[payment.AckStatus]struct {
		code     string
		terminal bool // VNPay stops retrying
	}{
		payment.AckProcessed:        {"00", true},
		payment.AckDuplicate:        {"02", true},
		payment.AckUnknownTxn:       {"01", false},
		payment.AckInvalidSignature: {"97", false},
		payment.AckInvalid:          {"99", false},
		payment.AckRetryLater:       {"99", false},
	}

	for ack, want := range cases {
		w := httptest.NewRecorder()
		p.AckNotification(w, ack)

		if w.Code != http.StatusOK {
			t.Fatalf("%v: HTTP %d, want 200 - VNPay reads the body, a 4xx reads as a transport failure", ack, w.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%v: body is not JSON: %v", ack, err)
		}
		if body["RspCode"] != want.code {
			t.Fatalf("%v: RspCode = %s, want %s", ack, body["RspCode"], want.code)
		}
		if body["Message"] == "" {
			t.Fatalf("%v: Message must not be empty", ack)
		}
		terminal := body["RspCode"] == "00" || body["RspCode"] == "02"
		if terminal != want.terminal {
			t.Fatalf("%v: RspCode %s stops retries = %v, want %v", ack, body["RspCode"], terminal, want.terminal)
		}
	}
}

/* -------------------------------------------------------------------------- */
/* Merchant API                                                                */
/* -------------------------------------------------------------------------- */

func TestQueryStatus_SignsWithThePipeOrderAndReadsBothStatusFields(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"vnp_ResponseCode":      "00",
			"vnp_TransactionStatus": "00",
			"vnp_Amount":            "14000000",
			"vnp_TransactionNo":     "14200001",
		})
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	txn := payment.Transaction{TxnRef: "CP260926TEST01", CreatedAt: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)}

	n, err := p.QueryStatus(context.Background(), txn)
	if err != nil {
		t.Fatalf("QueryStatus: %v", err)
	}
	if n.Status != payment.StatePaid || n.Amount != 140000 {
		t.Fatalf("got %s / %d, want paid / 140000", n.Status, n.Amount)
	}
	if gotBody["vnp_Command"] != "querydr" {
		t.Fatalf("command = %q", gotBody["vnp_Command"])
	}

	// The checksum must be the pipe hash in the documented order, not the query signer.
	want := pipeHash(testSecret,
		gotBody["vnp_RequestId"], "2.1.0", "querydr", "CINEMA01", "CP260926TEST01",
		gotBody["vnp_TransactionDate"], gotBody["vnp_CreateDate"], gotBody["vnp_IpAddr"],
		gotBody["vnp_OrderInfo"])
	if gotBody[fieldSecureHash] != want {
		t.Fatal("querydr checksum is not the documented pipe-joined hash")
	}
}

func TestQueryStatus_UnknownTransactionIsNotAFailedOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"vnp_ResponseCode": "91"})
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	_, err := p.QueryStatus(context.Background(), payment.Transaction{TxnRef: "NOPE", CreatedAt: time.Now()})
	if err != payment.ErrUnknownTxn {
		t.Fatalf("err = %v, want ErrUnknownTxn", err)
	}
}

func TestRefund_PartialUsesTransactionType03(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]string{"vnp_ResponseCode": "00"})
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	txn := payment.Transaction{
		TxnRef: "CP1", ProviderTxnID: "14200001",
		PaidAmount: 140000, CreatedAt: time.Now(),
	}

	// A partial refund sent as "02" refunds the WHOLE transaction, so the type must follow
	// the real figures rather than defaulting.
	if err := p.Refund(context.Background(), txn, payment.RefundRequest{Amount: 70000}); err != nil {
		t.Fatalf("partial refund: %v", err)
	}
	if gotBody["vnp_TransactionType"] != "03" {
		t.Fatalf("partial refund type = %q, want 03", gotBody["vnp_TransactionType"])
	}
	if gotBody["vnp_Amount"] != "7000000" {
		t.Fatalf("refund amount = %q, want 7000000 (70000 x100)", gotBody["vnp_Amount"])
	}

	if err := p.Refund(context.Background(), txn, payment.RefundRequest{Amount: 140000}); err != nil {
		t.Fatalf("full refund: %v", err)
	}
	if gotBody["vnp_TransactionType"] != "02" {
		t.Fatalf("full refund type = %q, want 02", gotBody["vnp_TransactionType"])
	}
}

func TestRefund_RefusalIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"vnp_ResponseCode": "99", "vnp_Message": "nope"})
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	err := p.Refund(context.Background(), payment.Transaction{TxnRef: "CP1", CreatedAt: time.Now()},
		payment.RefundRequest{Amount: 1000})
	if err == nil {
		t.Fatal("a refused refund must not report success")
	}
}

/* -------------------------------------------------------------------------- */
/* Construction                                                                */
/* -------------------------------------------------------------------------- */

func TestNew_RefusesIncompleteCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Options
	}{
		{"no tmn code", Options{HashSecret: "s", PayURL: "u"}},
		{"no secret", Options{TmnCode: "t", PayURL: "u"}},
		{"no pay url", Options{TmnCode: "t", HashSecret: "s"}},
	} {
		if _, err := New(tc.opts); err == nil {
			t.Fatalf("%s: New must refuse rather than come up unusable", tc.name)
		}
	}
}

// The simulator interface is what mounts a fake gateway under a public route. A real
// provider implementing it would expose an unauthenticated capture endpoint in production.
func TestProvider_IsNotASimulator(t *testing.T) {
	var p any = newTestProvider(t, "")
	if _, ok := p.(payment.Simulator); ok {
		t.Fatal("vnpay must not implement payment.Simulator")
	}
}
