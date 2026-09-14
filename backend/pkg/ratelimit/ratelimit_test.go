package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_BurstThenRefill(t *testing.T) {
	l := New(3, 1000)
	for i := 0; i < 3; i++ {
		if !l.Allow("ip") {
			t.Fatalf("request %d rejected inside the burst", i)
		}
	}
	if l.Allow("ip") {
		t.Fatal("request over the burst allowed")
	}
	if !l.Allow("other-ip") {
		t.Fatal("buckets are not per key")
	}
	time.Sleep(5 * time.Millisecond)
	if !l.Allow("ip") {
		t.Fatal("bucket did not refill")
	}
}

// T14: 5 consecutive failures lock the key out; success resets; old failures expire.
func TestFailureLimiter(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	l := NewFailureLimiter(5, time.Minute, func() time.Time { return now })

	for i := 1; i <= 4; i++ {
		if l.Fail("k") {
			t.Fatalf("locked after %d failures", i)
		}
	}
	if blocked, _ := l.Blocked("k"); blocked {
		t.Fatal("blocked before the 5th failure")
	}
	if !l.Fail("k") {
		t.Fatal("5th failure did not lock")
	}
	if blocked, left := l.Blocked("k"); !blocked || left != time.Minute {
		t.Fatalf("blocked=%v left=%v", blocked, left)
	}
	now = now.Add(time.Minute)
	if blocked, _ := l.Blocked("k"); blocked {
		t.Fatal("still blocked after the lockout")
	}

	for i := 0; i < 4; i++ {
		l.Fail("k")
	}
	l.Reset("k")
	if l.Fail("k") {
		t.Fatal("reset did not clear failures")
	}

	for i := 0; i < 4; i++ {
		l.Fail("slow")
	}
	now = now.Add(2 * time.Minute)
	if l.Fail("slow") {
		t.Fatal("failures older than the window still counted")
	}
}
