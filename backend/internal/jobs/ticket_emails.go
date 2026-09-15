package jobs

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/queue"
)

const emailChunk = 500

// NewSendTicketEmails mails confirmed bookings still without a ticket email. It retries
// failed sends and is the only path while the broker is down.
func NewSendTicketEmails(emails service.TicketEmailService) *batch.Job {
	return &batch.Job{
		Name:     "sendTicketEmails",
		Schedule: "0 */2 * * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			ids, err := emails.PendingIDs(ctx, emailChunk)
			if err != nil {
				return err
			}
			return batch.RunInChunks(ctx, opts, ids, func(ctx context.Context, id string) error {
				// Each email keeps its own retry schedule: no immediate retries.
				_, err := emails.Send(ctx, id)
				return batch.NoRetry(err)
			})
		},
	}
}

// ConsumeTicketEmails blocks until ctx is canceled. Every message is acked: a failed send
// is left to the sendTicketEmails cron instead of looping at the queue head.
func ConsumeTicketEmails(ctx context.Context, q *queue.Client, emails service.TicketEmailService) {
	err := q.Consume(ctx, service.TicketEmailQueue, func(ctx context.Context, body []byte) error {
		var msg struct {
			BookingID string `json:"booking_id"`
		}
		if err := json.Unmarshal(body, &msg); err != nil || msg.BookingID == "" {
			logger.Warn("dropping malformed ticket email message")
			return nil
		}
		if _, err := emails.Send(ctx, msg.BookingID); err != nil {
			logger.Warn("ticket email failed; cron will retry",
				logger.String("booking_id", msg.BookingID), logger.Err(err))
		}
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("ticket email consumer stopped", logger.Err(err))
	}
}
