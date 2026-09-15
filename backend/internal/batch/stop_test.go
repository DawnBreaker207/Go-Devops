package batch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type fakeRuns struct {
	startErr error

	mu       sync.Mutex
	finished []string
	orphans  int
	skipped  int
}

func (f *fakeRuns) Start(_ context.Context, name, by string) (*models.BatchJob, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	return &models.BatchJob{ID: "run-" + name, JobName: name, TriggeredBy: by}, nil
}

func (f *fakeRuns) Finish(_ context.Context, _ string, status, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finished = append(f.finished, status)
	return nil
}

func (f *fakeRuns) RecordProgress(context.Context, string, int, int) error { return nil }

func (f *fakeRuns) StopOrphans(context.Context) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orphans++
	return 1, nil
}

func (f *fakeRuns) RecordSkipped(context.Context, string, string, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.skipped++
	return nil
}

func (f *fakeRuns) List(context.Context, dto.PageQuery) ([]models.BatchJob, int64, error) {
	return nil, 0, nil
}

func (f *fakeRuns) RecentFailed(context.Context, time.Time, int) ([]models.BatchJob, error) {
	return nil, nil
}

func (f *fakeRuns) statuses() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.finished...)
}

func TestStopWaitsForRunningJob(t *testing.T) {
	m := NewManager(nil, &fakeRuns{})
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
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

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

func TestRun_PanicIsRecordedAndReleases(t *testing.T) {
	runs := &fakeRuns{}
	m := NewManager(nil, runs)
	var calls atomic.Int32
	m.Register(&Job{Name: "flaky", Run: func(context.Context, RunOptions) error {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		return nil
	}})
	if err := m.Run(context.Background(), "flaky", models.TriggerManual); err == nil {
		t.Fatal("panic reported no error")
	}
	if err := m.Run(context.Background(), "flaky", models.TriggerManual); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got := runs.statuses(); len(got) != 2 || got[0] != models.BatchFailed || got[1] != models.BatchSuccess {
		t.Fatalf("finished = %v, want [failed success]", got)
	}
}

func TestStart_MarksOrphanRunsStopped(t *testing.T) {
	runs := &fakeRuns{}
	m := NewManager(nil, runs)
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Stop(context.Background())
	if runs.orphans != 1 {
		t.Fatalf("StopOrphans calls = %d, want 1", runs.orphans)
	}
}

func TestStop_CancelsRunningJobContext(t *testing.T) {
	runs := &fakeRuns{}
	m := NewManager(nil, runs)
	running := make(chan struct{})
	m.Register(&Job{Name: "long", Run: func(ctx context.Context, _ RunOptions) error {
		close(running)
		<-ctx.Done()
		return ctx.Err()
	}})
	if _, err := m.Trigger("long", models.TriggerManual); err != nil {
		t.Fatal(err)
	}
	<-running

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	m.Stop(ctx)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Stop took %v", elapsed)
	}
	if got := runs.statuses(); len(got) != 1 || got[0] != models.BatchStopped {
		t.Fatalf("finished = %v, want [stopped]", got)
	}
}

func TestTrigger_UnknownJob(t *testing.T) {
	m := NewManager(nil, &fakeRuns{})
	if _, err := m.Trigger("nope", models.TriggerManual); !errors.Is(err, apperrors.ErrJobNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestCronConflict_WritesSkippedRow(t *testing.T) {
	runs := &fakeRuns{startErr: apperrors.ErrJobRunning}
	m := NewManager(nil, runs)
	m.Register(&Job{Name: "busy", Run: func(context.Context, RunOptions) error { return nil }})
	m.runScheduled("busy")
	if runs.skipped != 1 {
		t.Fatalf("skipped rows = %d, want 1", runs.skipped)
	}
}

func TestRunInChunks_NoRetrySkipsImmediately(t *testing.T) {
	calls := 0
	err := RunInChunks(context.Background(), RunOptions{}, []int{1}, func(context.Context, int) error {
		calls++
		return NoRetry(errors.New("has its own backoff"))
	})
	if err != nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want 1 call", err, calls)
	}
}
