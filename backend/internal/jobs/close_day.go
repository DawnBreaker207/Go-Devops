package jobs

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

// NewCloseDay writes daily aggregates at 23:59 local time. Yesterday is re-closed too so a
// missed run or late confirmation is caught up; the upsert keeps one row per day.
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
