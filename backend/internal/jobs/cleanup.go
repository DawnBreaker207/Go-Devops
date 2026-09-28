package jobs

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

const cleanupChunk = 5000

// Nightly 03:30 local: deletes past-retention audit logs, finished runs, dead tokens.
func NewCleanup(repo repository.MaintenanceRepository, retentionDays int, location *time.Location) *batch.Job {
	return &batch.Job{
		Name:     "cleanup",
		Schedule: "CRON_TZ=" + location.String() + " 0 30 3 * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			removed := 0
			steps := []func() (int64, error){
				func() (int64, error) { return repo.DeleteAuditOlderThan(ctx, retentionDays, cleanupChunk) },
				func() (int64, error) { return repo.DeleteFinishedRunsOlderThan(ctx, retentionDays, cleanupChunk) },
				func() (int64, error) { return repo.DeleteDeadRefreshTokens(ctx, cleanupChunk) },
				func() (int64, error) { return repo.DeleteDeadResetTokens(ctx, cleanupChunk) },
			}
			for _, step := range steps {
				for {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					n, err := step()
					if err != nil {
						return err
					}
					removed += int(n)
					if opts.Progress != nil && n > 0 {
						_ = opts.Progress(removed, 0)
					}
					if n < cleanupChunk {
						break
					}
				}
			}
			return nil
		},
	}
}
