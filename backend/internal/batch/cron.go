package batch

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Schedules use 6 fields (with seconds) so jobs can run every 30s.
type cronScheduler struct {
	cron *cron.Cron
}

func newCron() *cronScheduler {
	// Second net under Manager's recover: a panic in a cron func must not crash the process.
	return &cronScheduler{cron: cron.New(cron.WithSeconds(), cron.WithChain(cron.Recover(cronLogger{})))}
}

func (s *cronScheduler) addFunc(schedule string, fn func()) (cron.EntryID, error) {
	return s.cron.AddFunc(schedule, fn)
}

func (s *cronScheduler) start() { s.cron.Start() }

// The returned context is done once running jobs have finished.
func (s *cronScheduler) stop() context.Context { return s.cron.Stop() }

type cronLogger struct{}

func (cronLogger) Info(string, ...any) {}

func (cronLogger) Error(err error, msg string, keysAndValues ...any) {
	logger.Error("cron: "+msg, logger.Err(err), logger.String("details", fmt.Sprint(keysAndValues...)))
}
