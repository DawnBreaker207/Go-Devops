package jobs

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

// NewCloseDay builds the closeDay job (F21): at 23:59 local time it writes
// the daily_aggregates rows of yesterday and today. Yesterday is re-closed so
// a missed run or a late confirmation is caught up; the upsert keeps one row
// per day (E-B2). A manual run does the same.
func NewCloseDay(reports service.ReportService, location *time.Location) *batch.Job {
	return &batch.Job{
		Name:     "closeDay",
		Schedule: "CRON_TZ=" + location.String() + " 0 59 23 * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			now := time.Now().In(location)
			days := []time.Time{now.AddDate(0, 0, -1), now}
			for i, day := range days {
				if _, err := reports.CloseDay(ctx, day); err != nil {
					return err
				}
				if opts.Progress != nil {
					_ = opts.Progress(i+1, 0)
				}
			}
			return nil
		},
	}
}
