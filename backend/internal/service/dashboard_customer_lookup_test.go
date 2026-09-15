package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// AdminOverview: today's numbers are computed live (no closeDay run yet),
// and the 3 operational alert lists each pick up the row that should trip them.
func TestReport_AdminOverviewLiveTodayAndAlerts(t *testing.T) {
	e := newEnv(t)

	e.confirmed(e.users[0], "A1")

	stuckID := e.confirmed(e.users[1], "A2")
	e.must(e.db.Exec(`UPDATE payments SET status = 'refund_pending', refund_attempts = 9
		WHERE id = (SELECT payment_id FROM bookings WHERE id = ?)`, stuckID).Error)

	e.must(e.db.Exec(`INSERT INTO batch_jobs (id, job_name, status, started_at)
		VALUES (gen_random_uuid(), 'testjob', 'failed', NOW())`).Error)

	givenUpID := e.confirmed(e.users[2], "A3")
	e.must(e.db.Exec(`UPDATE bookings SET email_attempts = 6 WHERE id = ?`, givenUpID).Error)

	overview, err := e.reports.AdminOverview(e.ctx)
	e.must(err)

	if overview.Today.TicketsSold < 3 {
		t.Fatalf("today.tickets_sold = %d, want at least 3 (no closeDay run, computed live)", overview.Today.TicketsSold)
	}
	if overview.Today.TotalRevenue <= 0 {
		t.Fatalf("today.total_revenue = %d, want > 0", overview.Today.TotalRevenue)
	}

	if len(overview.Alerts.StuckRefunds) != 1 || overview.Alerts.StuckRefunds[0].Attempts != 9 {
		t.Fatalf("stuck refunds = %+v, want 1 row with attempts=9", overview.Alerts.StuckRefunds)
	}
	foundJob := false
	for _, j := range overview.Alerts.FailedJobs {
		if j.JobName == "testjob" {
			foundJob = true
		}
	}
	if !foundJob {
		t.Fatalf("failed jobs = %+v, missing testjob", overview.Alerts.FailedJobs)
	}
	foundEmail := false
	for _, em := range overview.Alerts.GivenUpEmails {
		if em.BookingID == givenUpID {
			foundEmail = true
		}
	}
	if !foundEmail {
		t.Fatalf("given-up emails = %+v, missing booking %s", overview.Alerts.GivenUpEmails, givenUpID)
	}
}

// StaffOverview composes the existing board + box office data and derives
// "awaiting check-in" as sold-minus-checked-in across the day's showtimes.
func TestReport_StaffOverviewAwaitingCheckin(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1", "A2")
	order, err := e.svc.Order(e.ctx, e.users[0], id)
	e.must(err)
	e.moveShowStart(e.showID, 10*time.Minute) // inside the check-in window
	if res, err := e.svc.Redeem(e.ctx, order.Tickets[0].Code, e.showID); err != nil || res.Status != models.RedeemOK {
		t.Fatalf("redeem: %+v %v", res, err)
	}

	overview, err := e.reports.StaffOverview(e.ctx, "")
	e.must(err)
	if overview.AwaitingCheckin != 1 {
		t.Fatalf("awaiting_checkin = %d, want 1 (2 sold, 1 redeemed)", overview.AwaitingCheckin)
	}
}

// AdminOrder is Order without the ownership check: a booking's owner sees it
// via Order, but another user's own Order call is refused (403) while
// AdminOrder (support lookup) succeeds for anyone.
func TestBooking_AdminOrderBypassesOwnership(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1")

	if _, err := e.svc.Order(e.ctx, e.users[1], id); httpStatus(err) != http.StatusForbidden {
		t.Fatalf("another user's Order call = %v, want 403", err)
	}

	admin, err := e.svc.AdminOrder(e.ctx, id)
	e.must(err)
	if admin.ID != id {
		t.Fatalf("AdminOrder id = %s, want %s", admin.ID, id)
	}
}

// The customer-lookup endpoints (support tool) reuse UserService/BookingService
// as-is; this proves List(role=customer) + GetByID + List(userID) + AdminOrder
// together answer exactly what the staff handlers expose.
func TestCustomerLookup_ProfileHistoryAndOrderDetail(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "lookup-target@test.local", "secret123")
	bookingID := e.confirmed(u.ID, "A1")

	list, total, err := e.accounts.List(e.ctx, dto.UserListQuery{
		PageQuery: dto.PageQuery{Page: 1, PageSize: 50}, Role: models.RoleCustomer,
	})
	e.must(err)
	found := false
	for _, row := range list {
		if row.ID == u.ID {
			found = true
		}
	}
	if !found || total == 0 {
		t.Fatalf("customer %s not found in role=customer list", u.ID)
	}

	profile, err := e.accounts.GetByID(e.ctx, u.ID)
	e.must(err)
	if profile.Email != u.Email {
		t.Fatalf("profile email = %s, want %s", profile.Email, u.Email)
	}

	orders, total, err := e.svc.List(e.ctx, u.ID, dto.PageQuery{Page: 1, PageSize: 10})
	e.must(err)
	if total != 1 || len(orders) != 1 || orders[0].ID != bookingID {
		t.Fatalf("customer orders = %+v (total %d), want 1 row for %s", orders, total, bookingID)
	}

	detail, err := e.svc.AdminOrder(e.ctx, bookingID)
	e.must(err)
	if detail.ID != bookingID {
		t.Fatalf("AdminOrder id = %s, want %s", detail.ID, bookingID)
	}
}

// A non-customer id (staff/admin) must 404 through the support-lookup path.
// The service layer doesn't restrict GetByID by role — the staff handler
// does, by checking the returned role and answering apperrors.ErrUserNotFound
// itself — so this documents the role the handler branches on.
func TestCustomerLookup_NonCustomerRoleIsDistinguishable(t *testing.T) {
	e := newEnv(t)
	staff := e.newUser(models.RoleStaff, "not-a-customer@test.local", "secret123")
	profile, err := e.accounts.GetByID(e.ctx, staff.ID)
	e.must(err)
	if profile.Role == models.RoleCustomer {
		t.Fatalf("expected a non-customer role, got %s", profile.Role)
	}
}
