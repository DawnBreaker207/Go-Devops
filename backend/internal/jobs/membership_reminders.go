package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/notify"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

// NewMembershipReminders emails a member 3-7 days before their membership
// expires (ADVANCED_FEATURES_DISCUSSION.md Phần 1.2). Membership renewal is
// manual (no auto-charge), so this is the only nudge a member gets; sending
// it once is tracked via UserMembership.ReminderSentAt so a daily run does
// not resend the same email for up to 5 days.
func NewMembershipReminders(memberships *repository.MembershipRepository, users repository.UserRepository, mailer notify.Mailer, location *time.Location) *batch.Job {
	return &batch.Job{
		Name:     "membershipReminders",
		Schedule: "CRON_TZ=" + location.String() + " 0 0 8 * * *",
		Run: func(ctx context.Context, opts batch.RunOptions) error {
			now := time.Now()
			rows, err := memberships.ExpiringSoon(ctx, now.AddDate(0, 0, 3), now.AddDate(0, 0, 7))
			if err != nil {
				return err
			}
			return batch.RunInChunks(ctx, opts, rows, func(ctx context.Context, m repository.UserMembershipRow) error {
				user, err := users.FindByID(ctx, m.UserID)
				if err != nil {
					return err
				}
				html := fmt.Sprintf(
					"<p>Xin chào %s,</p><p>Gói hội viên của bạn sẽ hết hạn vào %s. Gia hạn ngay để không gián đoạn ưu đãi.</p>",
					user.FullName, m.ExpiresAt.In(location).Format("02/01/2006"))
				if err := mailer.Send(ctx, notify.Message{To: user.Email, Subject: "Gói hội viên sắp hết hạn", HTML: html}); err != nil {
					return batch.NoRetry(err) // a bad address will never send; do not burn 3 retries on it
				}
				return memberships.MarkReminderSent(ctx, m.ID)
			})
		},
	}
}
