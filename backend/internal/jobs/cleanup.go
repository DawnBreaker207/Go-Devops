package jobs

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

const cleanupChunk = 5000

// NewCleanup builds the cleanup job: every night at 03:30 local time it
// deletes audit logs older than the retention (kept >= 90 days, FR-AUDIT-02,
// T40) and refresh tokens dead for a day, chunk by chunk.
func NewCleanup(repo repository.MaintenanceRepository, auditRetentionDays int, location *time.Location) *batch.Job {
	return &batch.Job{
		Name:     "cleanup",
		Schedule: "CRON_TZ=" + location.String() + " 0 30 3 * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			removed := 0
			steps := []func() (int64, error){
				func() (int64, error) { return repo.DeleteAuditOlderThan(ctx, auditRetentionDays, cleanupChunk) },
				func() (int64, error) { return repo.DeleteDeadRefreshTokens(ctx, cleanupChunk) },
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
