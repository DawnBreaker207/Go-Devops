package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// T39 (FR-AUDIT-01): every way a booking leaves PENDING — confirm, refund,
// replaced by a new hold, customer cancel, sweep — and every payment state
// change leaves an audit row written with it.
func TestAudit_EveryStateChangeIsLogged(t *testing.T) {
	e := newEnv(t)
	confirmed := e.confirmed(e.users[0], "A1")

	refunded := e.mustHold(e.users[1], "A2")
	refundRef := e.pay(e.users[1], refunded.BookingID)
	e.shiftHold(refunded.BookingID, -1)
	e.capture(refundRef)
	if ack := e.ipn(refundRef); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}

	replaced := e.mustHold(e.users[2], "A3")
	e.mustHold(e.users[2], "A4")

	canceled := e.mustHold(e.users[3], "B1")
	_, err := e.svc.Cancel(e.ctx, e.users[3], canceled.BookingID)
	e.must(err)

	swept := e.mustHold(e.users[4], "B2")
	sweptRef := e.pay(e.users[4], swept.BookingID)
	e.shiftHold(swept.BookingID, -1)
	e.must(e.db.Exec(`UPDATE payments SET created_at = NOW() - INTERVAL '3 minutes',
		expires_at = NOW() - INTERVAL '20 minutes' WHERE txn_ref = ?`, sweptRef).Error)
	_, err = e.svc.SweepExpired(e.ctx, 500)
	e.must(err)

	failing := e.mustHold(e.users[5], "B3")
	e.gw.SetDown(true)
	if _, err := e.svc.Pay(e.ctx, e.users[5], failing.BookingID, dto.PayRequest{Provider: "mock"}); httpStatus(err) != http.StatusBadGateway {
		t.Fatalf("gateway down: err = %v", err)
	}
	e.gw.SetDown(false)
	var failedPayment string
	e.must(e.db.Raw(`SELECT id FROM payments WHERE booking_id = ?`, failing.BookingID).Scan(&failedPayment).Error)

	want := map[string]string{
		confirmed:               "orders.confirm",
		refunded.BookingID:      "orders.refund",
		replaced.BookingID:      "orders.expire",
		canceled.BookingID:      "orders.cancel",
		swept.BookingID:         "orders.expire",
		e.payment(refundRef).ID: "payments.refunded",
		e.payment(sweptRef).ID:  "payments.abandoned",
		failedPayment:           "payments.create_failed",
	}
	for id, action := range want {
		if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = ? AND action = ?`, id, action); n != 1 {
			t.Errorf("%s rows for %s = %d, want 1", action, id, n)
		}
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE action = 'seats.release_expired' AND resource_id = ?`, e.showID); n < 1 {
		t.Error("sweep seat release not logged")
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings b WHERE b.status <> 'pending' AND NOT EXISTS (
		SELECT 1 FROM audit_logs a WHERE a.resource_id = b.id::text
		AND a.action IN ('orders.confirm', 'orders.refund', 'orders.expire', 'orders.cancel'))`); n != 0 {
		t.Errorf("%d bookings left PENDING without an audit row", n)
	}
	e.checkInvariants()
}

// T40 (FR-AUDIT-02): cleanup deletes audit rows past the retention only, and
// refresh tokens dead for a day.
func TestCleanupJob_RemovesOnlyExpiredData(t *testing.T) {
	e := newEnv(t)
	insertAudit := func(action string, daysAgo int) {
		e.must(e.db.Exec(`INSERT INTO audit_logs (id, action, resource_type, outcome, created_at)
			VALUES (gen_random_uuid(), ?, 'test', 'success', NOW() - make_interval(days => CAST(? AS int)))`, action, daysAgo).Error)
	}
	for i := 0; i < 3; i++ {
		insertAudit("test.old", 91)
	}
	insertAudit("test.recent", 89)
	insertAudit("test.today", 0)
	u := e.users[0]
	e.must(e.db.Exec(`INSERT INTO refresh_tokens (id, user_id, family_id, expires_at) VALUES
		(gen_random_uuid(), ?, gen_random_uuid(), NOW() - INTERVAL '2 days'),
		(gen_random_uuid(), ?, gen_random_uuid(), NOW() + INTERVAL '2 days')`, u, u).Error)

	processed, _ := e.runJob(jobs.NewCleanup(repository.NewMaintenanceRepository(e.db), 90, time.UTC))
	if processed != 4 {
		t.Fatalf("removed = %d, want 4 (3 audit rows + 1 token)", processed)
	}
	for action, want := range map[string]int64{"test.old": 0, "test.recent": 1, "test.today": 1} {
		if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE action = ?`, action); n != want {
			t.Errorf("%s rows = %d, want %d", action, n, want)
		}
	}
	if n := e.count(`SELECT COUNT(*) FROM refresh_tokens`); n != 1 {
		t.Errorf("refresh tokens = %d, want 1", n)
	}
}

// T44 (FR-BATCH-01) / E-B6: every run gets its own batch_jobs row with its
// counts, and a second run while one is RUNNING is refused.
func TestBatch_EveryRunLogged(t *testing.T) {
	e := newEnv(t)
	manager := batch.NewManager(e.db, repository.NewBatchJobRepository(e.db))
	manager.Register(jobs.NewSweepExpiredHolds(e.svc))

	h := e.mustHold(e.users[0], "A1", "A2")
	e.shiftHold(h.BookingID, -1)
	for i := 0; i < 2; i++ {
		e.must(manager.Run(e.ctx, "sweepExpiredHolds", models.TriggerManual))
	}
	var runs []models.BatchJob
	e.must(e.db.Where("job_name = ?", "sweepExpiredHolds").Order("started_at").Find(&runs).Error)
	if len(runs) != 2 {
		t.Fatalf("runs = %d, want 2", len(runs))
	}
	if runs[0].Status != models.BatchSuccess || runs[0].ProcessedRows != 3 ||
		runs[1].Status != models.BatchSuccess || runs[1].ProcessedRows != 0 || runs[1].FinishedAt == nil {
		t.Fatalf("runs = %+v", runs)
	}

	e.must(e.db.Exec(`INSERT INTO batch_jobs (id, job_name, triggered_by, status)
		VALUES (gen_random_uuid(), 'sweepExpiredHolds', 'manual', 'running')`).Error)
	if err := manager.Run(e.ctx, "sweepExpiredHolds", models.TriggerManual); !isAppErr(err, apperrors.ErrJobRunning) {
		t.Fatalf("concurrent run: err = %v", err)
	}
}
