package service_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

// T23 / E-ML1 / E-ML2: one email with a QR per ticket; a broken mailer never touches the sale and is retried.
func TestTicketEmails(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	id := e.confirmed(u, "A1", "B1")
	order, err := e.svc.Order(e.ctx, u, id)
	e.must(err)
	job := jobs.NewSendTicketEmails(e.emails)

	sent, failed := e.runJob(job)
	if sent != 1 || failed != 0 {
		t.Fatalf("sent=%d failed=%d, want 1/0", sent, failed)
	}
	msgs := e.mailer.Sent()
	if len(msgs) != 1 || msgs[0].To != e.emailOf[u] {
		t.Fatalf("messages = %+v", msgs)
	}
	html := msgs[0].HTML
	if n := strings.Count(html, `src="data:image/png;base64,`); n != 2 {
		t.Fatalf("QR images = %d, want 2", n)
	}
	for _, tk := range order.Tickets {
		if !strings.Contains(html, tk.Code) || !strings.Contains(html, "Ghế "+tk.SeatLabel) {
			t.Fatalf("email misses ticket %s", tk.SeatLabel)
		}
	}

	// Already sent: queue redelivery or the cron sends nothing more.
	if ok, err := e.emails.Send(e.ctx, id); ok || err != nil {
		t.Fatalf("resend: ok=%v err=%v", ok, err)
	}
	if sent, _ := e.runJob(job); sent != 0 {
		t.Fatalf("second run sent %d", sent)
	}

	id2 := e.confirmed(e.users[1], "B2")
	e.mailer.FailWith(errors.New("smtp unavailable"))
	sent, failed = e.runJob(job)
	if sent != 0 || failed != 1 {
		t.Fatalf("while down: sent=%d failed=%d", sent, failed)
	}
	b := e.wantStatus(id2, models.BookingConfirmed)
	if b.EmailSentAt != nil || b.EmailAttempts != 1 || b.EmailClaimedUntil == nil || !b.EmailClaimedUntil.After(time.Now()) {
		t.Fatalf("after a failed send: sent_at=%v attempts=%d claimed_until=%v", b.EmailSentAt, b.EmailAttempts, b.EmailClaimedUntil)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = 'email.ticket_failed'`, id2); n != 0 {
		t.Fatalf("email failure audited before its tries ran out: %d rows", n)
	}

	e.mailer.FailWith(nil)
	if sent, _ := e.runJob(job); sent != 0 {
		t.Fatalf("retried before its backoff: sent %d", sent)
	}
	e.must(e.db.Exec(`UPDATE bookings SET email_claimed_until = NOW() - INTERVAL '1 second' WHERE id = ?`, id2).Error)
	if sent, _ := e.runJob(job); sent != 1 {
		t.Fatalf("retry sent %d, want 1", sent)
	}
	if len(e.mailer.Sent()) != 2 {
		t.Fatalf("total messages = %d, want 2", len(e.mailer.Sent()))
	}
}

func TestTicketEmails_FailureBacksOffAndCaps(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1")
	e.mailer.FailWith(errors.New("smtp unavailable"))
	for i := 1; i <= repository.MaxTicketEmailAttempts; i++ {
		e.must(e.db.Exec(`UPDATE bookings SET email_claimed_until = NULL WHERE id = ?`, id).Error)
		if sent, err := e.emails.Send(e.ctx, id); sent || err == nil {
			t.Fatalf("try %d: sent=%v err=%v", i, sent, err)
		}
	}
	e.must(e.db.Exec(`UPDATE bookings SET email_claimed_until = NULL WHERE id = ?`, id).Error)
	if sent, err := e.emails.Send(e.ctx, id); sent || err != nil {
		t.Fatalf("try past the cap: sent=%v err=%v, want nothing done", sent, err)
	}
	ids, err := e.emails.PendingIDs(e.ctx, 10)
	e.must(err)
	if len(ids) != 0 {
		t.Fatalf("given-up email still pending: %v", ids)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = 'email.ticket_failed'`, id); n != 1 {
		t.Fatalf("failure audit rows = %d, want 1", n)
	}
	e.wantStatus(id, models.BookingConfirmed)
}

func TestTicketEmails_ExpiredClaimIsRetried(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1")
	e.must(e.db.Exec(`UPDATE bookings SET email_attempts = 1, email_claimed_until = NOW() + INTERVAL '4 minutes' WHERE id = ?`, id).Error)
	if sent, err := e.emails.Send(e.ctx, id); sent || err != nil {
		t.Fatalf("while leased: sent=%v err=%v", sent, err)
	}
	e.must(e.db.Exec(`UPDATE bookings SET email_claimed_until = NOW() - INTERVAL '1 second' WHERE id = ?`, id).Error)
	if sent, err := e.emails.Send(e.ctx, id); !sent || err != nil {
		t.Fatalf("after the lease: sent=%v err=%v", sent, err)
	}
	if b := e.booking(id); b.EmailSentAt == nil || b.EmailClaimedUntil != nil {
		t.Fatalf("sent_at=%v claimed_until=%v", b.EmailSentAt, b.EmailClaimedUntil)
	}
}
