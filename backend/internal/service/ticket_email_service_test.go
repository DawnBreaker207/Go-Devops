package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// T23 / E-ML1 / E-ML2: a confirmed booking gets one email with a QR per
// ticket; a broken mailer never touches the sale, the job retries then skips,
// and the next run sends it.
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

	// Mail server down.
	id2 := e.confirmed(e.users[1], "B2")
	e.mailer.FailWith(errors.New("smtp unavailable"))
	sent, failed = e.runJob(job)
	if sent != 0 || failed != 1 {
		t.Fatalf("while down: sent=%d failed=%d", sent, failed)
	}
	if b := e.wantStatus(id2, models.BookingConfirmed); b.EmailSentAt != nil {
		t.Fatal("failed send left email_sent_at set")
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = 'email.ticket_failed' AND outcome = 'failure'`, id2); n != batch.ItemAttempts {
		t.Fatalf("email failure audit rows = %d, want one per attempt (%d)", n, batch.ItemAttempts)
	}

	e.mailer.FailWith(nil)
	if sent, _ := e.runJob(job); sent != 1 {
		t.Fatalf("retry sent %d, want 1", sent)
	}
	if len(e.mailer.Sent()) != 2 {
		t.Fatalf("total messages = %d, want 2", len(e.mailer.Sent()))
	}
}
