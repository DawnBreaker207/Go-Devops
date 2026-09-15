package batch

import (
	"context"
	"errors"
	"testing"
)

// E-B4: a flaky item succeeds on retry, a broken one is skipped, the job goes on.
func TestRunInChunks_RetryThenSkip(t *testing.T) {
	calls := map[int]int{}
	items := make([]int, 1203)
	for i := range items {
		items[i] = i
	}
	var lastProcessed, lastSkipped, progressCalls int
	opts := RunOptions{Progress: func(p, s int) error {
		lastProcessed, lastSkipped = p, s
		progressCalls++
		return nil
	}}

	err := RunInChunks(context.Background(), opts, items, func(_ context.Context, item int) error {
		calls[item]++
		switch {
		case item == 7 && calls[item] < ItemAttempts:
			return errors.New("temporary")
		case item == 900:
			return errors.New("permanent")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if lastProcessed != 1202 || lastSkipped != 1 {
		t.Fatalf("processed=%d skipped=%d", lastProcessed, lastSkipped)
	}
	if calls[7] != ItemAttempts || calls[900] != ItemAttempts || calls[8] != 1 {
		t.Fatalf("attempts: flaky=%d broken=%d normal=%d", calls[7], calls[900], calls[8])
	}
	if progressCalls != 3 {
		t.Fatalf("progress after %d chunks, want 3", progressCalls)
	}
}
