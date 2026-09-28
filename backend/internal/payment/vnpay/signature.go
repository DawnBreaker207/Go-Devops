// Package vnpay adapts VNPay's hosted checkout to the gateway-agnostic payment.Provider
// contract. It ships no simulator: a real provider must not have one.
package vnpay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// Fields VNPay excludes from the signed string. vnp_SecureHash is the signature itself;
// vnp_SecureHashType is metadata that older integrations still send.
const (
	fieldSecureHash     = "vnp_SecureHash"
	fieldSecureHashType = "vnp_SecureHashType"
)

// signData builds VNPay's checksum source and HMAC-SHA512s it with the merchant secret.
//
// The rules, from VNPay's own docs, and every one of them is a place integrations break:
//
//   - parameters are sorted ASCENDING BY NAME, not by insertion order;
//   - both name and value are URL-ENCODED before they go into the string. Go's
//     url.QueryEscape matches PHP's urlencode, including encoding a space as "+",
//     which is what VNPay's reference implementation produces;
//   - vnp_SecureHash and vnp_SecureHashType are excluded;
//   - the result is lowercase hex.
//
// The asymmetric version of this - signing encoded but verifying raw, or the reverse -
// is the classic failure, so signing and verifying share this one function.
func signData(secret string, params map[string]string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(signSource(params)))
	return hex.EncodeToString(mac.Sum(nil))
}

// signSource builds the exact string that gets hashed. Split out from signData so the
// tests can assert the ENCODING RULES themselves rather than comparing one HMAC to
// another, which would pass just as happily with both sides wrong.
func signSource(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == fieldSecureHash || k == fieldSecureHashType {
			continue
		}
		// VNPay omits empty values rather than signing "key=".
		if params[k] == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(params[k]))
	}
	return b.String()
}

// verify re-signs the parameters and compares in constant time.
//
// VNPay's hex is uppercase in some SDKs and lowercase in others, so the comparison is
// case-insensitive on the hex - which changes nothing about its strength, because both
// sides are normalised before a constant-time compare of the same byte length.
func verify(secret string, params map[string]string, given string) bool {
	if given == "" {
		return false
	}
	want := signData(secret, params)
	return hmac.Equal([]byte(strings.ToLower(want)), []byte(strings.ToLower(given)))
}

// queryParams flattens a URL query to the single-valued map the signature works on.
// A repeated parameter keeps its first value, matching VNPay's own parsing.
func queryParams(q url.Values) map[string]string {
	out := make(map[string]string, len(q))
	for k, v := range q {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}
