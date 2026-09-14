package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// T51 / E-T4..E-T6: outside the check-in window the gate answers too_early or
// closed and the ticket stays ISSUED; inside it the ticket gets in once, even
// after the showtime closed for sales. Refused scans are audited without the
// scanned code.
func TestRedeem_CheckinWindow(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1") // the show starts in 3 hours
	order, err := e.svc.Order(e.ctx, e.users[0], id)
	e.must(err)
	ticket := order.Tickets[0]

	scan := func(want string) *dto.RedeemResponse {
		t.Helper()
		res, err := e.svc.Redeem(e.ctx, ticket.Code, e.showID)
		e.must(err)
		if res.Status != want {
			t.Fatalf("redeem = %s, want %s", res.Status, want)
		}
		return res
	}
	wantTicket := func(status string) {
		t.Helper()
		if n := e.count(`SELECT COUNT(*) FROM tickets WHERE id = ? AND status = ?`, ticket.ID, status); n != 1 {
			t.Fatalf("ticket is not %s", status)
		}
	}

	res := scan(models.RedeemTooEarly)
	if res.StartAt == nil || res.CheckinOpensAt == nil || res.CheckinClosesAt == nil ||
		!res.CheckinOpensAt.Equal(res.StartAt.Add(-30*time.Minute)) ||
		!res.CheckinClosesAt.Equal(res.StartAt.Add(20*time.Minute)) {
		t.Fatalf("window %v .. %v around start %v", res.CheckinOpensAt, res.CheckinClosesAt, res.StartAt)
	}
	e.moveShowStart(e.showID, 31*time.Minute)
	scan(models.RedeemTooEarly)
	e.moveShowStart(e.showID, -21*time.Minute)
	scan(models.RedeemClosed)
	wantTicket(models.TicketIssued)

	// E-T5: inside the window a showtime closed for sales still lets its tickets in.
	e.moveShowStart(e.showID, 25*time.Minute)
	e.must(e.db.Exec(`UPDATE showtimes SET status = 'closed' WHERE id = ?`, e.showID).Error)
	scan(models.RedeemOK)
	wantTicket(models.TicketRedeemed)
	// A used ticket reads "used" even once the doors closed.
	e.moveShowStart(e.showID, -30*time.Minute)
	scan(models.RedeemUsed)

	var verdicts []string
	e.must(e.db.Raw(`SELECT error_message FROM audit_logs WHERE action = 'staff.redeem_ticket'
		AND outcome = 'failure' AND resource_id = ? ORDER BY created_at`, ticket.ID).Scan(&verdicts).Error)
	if strings.Join(verdicts, ",") != "too_early,too_early,closed,used" {
		t.Fatalf("audited refusals = %v", verdicts)
	}

	if res, err := e.svc.Redeem(e.ctx, "NO-SUCH-CODE", e.showID); err != nil || res.Status != models.RedeemNotFound {
		t.Fatalf("unknown code: %+v %v", res, err)
	}
	if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE error_message = 'not_found' AND COALESCE(resource_id, '') = ''`); n != 1 {
		t.Fatalf("not_found audit rows = %d, want 1", n)
	}
	for _, code := range []string{ticket.Code, "NO-SUCH-CODE"} {
		if n := e.count(`SELECT COUNT(*) FROM audit_logs WHERE COALESCE(after_json::text, '') LIKE ?
			OR COALESCE(resource_id, '') = ?`, "%"+code+"%", code); n != 0 {
			t.Fatalf("%d audit rows carry the scanned code %s", n, code)
		}
	}
}
