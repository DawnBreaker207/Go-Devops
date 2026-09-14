// Package jobs hosts background jobs registered into the batch manager.
package jobs

import (
	"context"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

const sweepChunk = 500

type holdSweeper interface {
	SweepExpired(ctx context.Context, limit int) (service.SweepResult, error)
}

// NewSweepExpiredHolds also expires unpaid bookings, settles stuck paid bookings and
// retries failed refunds, looping until caught up.
func NewSweepExpiredHolds(sweeper holdSweeper) *batch.Job {
	return &batch.Job{
		Name:     "sweepExpiredHolds",
		Schedule: "*/30 * * * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			processed := 0
			for {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				res, err := sweeper.SweepExpired(ctx, sweepChunk)
				processed += res.Total()
				if opts.Progress != nil && res.Total() > 0 {
					_ = opts.Progress(processed, 0)
				}
				if err != nil {
					return err
				}
				if res.ReleasedSeats < sweepChunk && res.ExpiredBookings < sweepChunk {
					return nil
				}
			}
		},
	}
}
