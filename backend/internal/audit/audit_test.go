package audit

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// H7: values are cut by characters, never splitting a multi-byte one.
func TestTruncate_RuneSafe(t *testing.T) {
	long := strings.Repeat("ệ", 600)
	got := truncate(long, 512)
	if n := utf8.RuneCountInString(got); n != 512 || !utf8.ValidString(got) {
		t.Fatalf("truncate = %d runes, valid=%v", n, utf8.ValidString(got))
	}
	if got := truncate("abc", 5); got != "abc" {
		t.Fatalf("short value changed: %q", got)
	}
}
