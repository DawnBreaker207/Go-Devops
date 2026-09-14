package batch

import (
	"github.com/robfig/cron/v3"
)

// cronScheduler wraps robfig/cron so the Manager doesn't depend on the
// scheduler library directly. Schedules use 6 fields (seconds) to allow
// 30-second sweeps.
type cronScheduler struct {
	cron *cron.Cron
}

func newCron() *cronScheduler {
	return &cronScheduler{cron: cron.New(cron.WithSeconds())}
}

func (s *cronScheduler) addFunc(schedule string, fn func()) (cron.EntryID, error) {
	return s.cron.AddFunc(schedule, fn)
}

func (s *cronScheduler) start() { s.cron.Start() }

func (s *cronScheduler) stop() { s.cron.Stop() }