package batch

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// cronScheduler wraps robfig/cron so the Manager doesn't depend on the
// scheduler library directly. Schedules use 6 fields (seconds) to allow
// 30-second sweeps.
type cronScheduler struct {
	cron *cron.Cron
}

func newCron() *cronScheduler {
	// Recover is a second net under Manager's own recover: a panic in a
	// scheduled func must never take the whole process down.
	return &cronScheduler{cron: cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cronLogger{})))}
}

func (s *cronScheduler) addFunc(schedule string, fn func()) (cron.EntryID, error) {
	return s.cron.AddFunc(schedule, fn)
}

func (s *cronScheduler) start() { s.cron.Start() }

// stop halts scheduling; the context is done once running jobs finished.
func (s *cronScheduler) stop() context.Context { return s.cron.Stop() }

// cronLogger forwards the scheduler's error lines (recovered panics) to zap.
type cronLogger struct{}

func (cronLogger) Info(string, ...any) {}

func (cronLogger) Error(err error, msg string, keysAndValues ...any) {
	logger.Error("cron: "+msg, logger.Err(err), logger.String("details", fmt.Sprint(keysAndValues...)))
}
