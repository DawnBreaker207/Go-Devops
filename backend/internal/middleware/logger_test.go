package middleware

import (
	"strings"
	"testing"
)

func TestRedactQuery(t *testing.T) {
	got := redactQuery("token=abc123&date=2026-09-14&sig=deadbeef&signature=s1&ref=CP1")
	for _, secret := range []string{"abc123", "deadbeef", "s1&"} {
		if strings.Contains(got+"&", secret) {
			t.Fatalf("secret %q leaked: %s", secret, got)
		}
	}
	for _, kept := range []string{"date=2026-09-14", "ref=CP1", "token=REDACTED"} {
		if !strings.Contains(got, kept) {
			t.Fatalf("%q missing: %s", kept, got)
		}
	}
	if redactQuery("") != "" {
		t.Fatal("empty query changed")
	}
}
