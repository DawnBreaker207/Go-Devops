package batch

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type fakeRuns struct{}

func (fakeRuns) Start(context.Context, string, string) (*models.BatchJob, error) {
	return &models.BatchJob{ID: "run"}, nil
}
func (fakeRuns) Finish(context.Context, string, string, string) error   { return nil }
func (fakeRuns) RecordProgress(context.Context, string, int, int) error { return nil }
func (fakeRuns) List(context.Context, dto.PageQuery) ([]models.BatchJob, int64, error) {
	return nil, 0, nil
}

// Graceful shutdown waits for a running cron job before the DB is closed.
func TestStopWaitsForRunningJob(t *testing.T) {
	m := NewManager(nil, fakeRuns{})
	started := make(chan struct{}, 1)
	var finished atomic.Bool
	m.Register(&Job{Name: "slow", Schedule: "* * * * * *", Run: func(context.Context, RunOptions) error {
		select {
		case started <- struct{}{}:
		default:
		}
		time.Sleep(700 * time.Millisecond)
		finished.Store(true)
		return nil
	}})
	m.Start()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("job never started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.Stop(ctx)
	if !finished.Load() {
		t.Fatal("Stop returned while the job was still running")
	}
}
