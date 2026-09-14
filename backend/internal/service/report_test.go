package service_test

import (
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
)

// T24 / E-B2 / E-D4: closing a day twice keeps one row with the same numbers; only confirmed money counts.
func TestCloseDay_IdempotentAndConfirmedOnly(t *testing.T) {
	e := newEnv(t)
	confirmed := e.confirmed(e.users[0], "A1", "B1")
	e.mustHold(e.users[1], "A2") // pending: not revenue
	refunded := e.mustHold(e.users[2], "A3")
	ref := e.pay(e.users[2], refunded.BookingID)
	e.capture(ref, mock.CaptureOptions{AmountDelta: 1000})
	if ack := e.ipn(ref); ack != payment.AckProcessed {
		t.Fatalf("ack = %s", ack)
	}
	e.wantStatus(refunded.BookingID, models.BookingRefunded)

	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	paidAt := *e.booking(confirmed).PaidAt
	days := map[string]time.Time{
		paidAt.UTC().Format(dto.DateLayout):       paidAt,
		show.StartAt.UTC().Format(dto.DateLayout): show.StartAt,
	}

	type totals struct {
		revenue                 int64
		tickets, sold, capacity int
		showtimesInBreakdown    int
	}
	var runs []totals
	for round := 0; round < 2; round++ {
		var tot totals
		for _, day := range days {
			agg, err := e.reports.CloseDay(e.ctx, day)
			e.must(err)
			tot.revenue += agg.TotalRevenue
			tot.tickets += agg.TicketsSold
			tot.sold += agg.SeatsSold
			tot.capacity += agg.Capacity
			if list, ok := agg.Breakdown["showtimes"].([]any); ok {
				tot.showtimesInBreakdown += len(list)
			}
		}
		runs = append(runs, tot)
	}
	if runs[0] != runs[1] {
		t.Fatalf("second close changed numbers: %+v vs %+v", runs[0], runs[1])
	}
	if n := e.count(`SELECT COUNT(*) FROM daily_aggregates`); n != int64(len(days)) {
		t.Fatalf("rows = %d, want %d (one per day)", n, len(days))
	}
	got := runs[1]
	want := totals{revenue: priceStandard + priceVIP, tickets: 2, sold: 2, capacity: 9, showtimesInBreakdown: 1}
	if got != want {
		t.Fatalf("totals = %+v, want %+v", got, want)
	}

	// I4: stored revenue equals confirmed money by payment time.
	if n := e.count(`SELECT COUNT(*) FROM daily_aggregates d WHERE d.total_revenue <> COALESCE((
		SELECT SUM(b.total_amount) FROM bookings b WHERE b.status = 'confirmed'
		  AND b.paid_at >= (d.report_date::timestamp AT TIME ZONE 'UTC')
		  AND b.paid_at < ((d.report_date + 1)::timestamp AT TIME ZONE 'UTC')), 0)`); n != 0 {
		t.Fatalf("I4 broken on %d days", n)
	}

	// T15 (service side) / E-D1: the admin report reads the same numbers.
	dates := make([]string, 0, len(days))
	for d := range days {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	report, err := e.reports.DailyReport(e.ctx, dates[0], dates[len(dates)-1])
	e.must(err)
	if report.TotalRevenue != got.revenue || report.TicketsSold != got.tickets || len(report.Days) != len(days) {
		t.Fatalf("report = %+v", report)
	}
	if _, err := e.reports.DailyReport(e.ctx, "2026-09-15", "2026-09-14"); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("from after to: err = %v", err)
	}
}

// The closeDay job closes yesterday and today.
func TestCloseDayJob(t *testing.T) {
	e := newEnv(t)
	if processed, _ := e.runJob(jobs.NewCloseDay(e.reports, time.UTC)); processed != 2 {
		t.Fatalf("processed = %d, want 2 days", processed)
	}
	if n := e.count(`SELECT COUNT(*) FROM daily_aggregates`); n != 2 {
		t.Fatalf("rows = %d, want 2", n)
	}
}

// Seat counts per showtime of the day and the gate list.
func TestStaffBoard(t *testing.T) {
	e := newEnv(t)
	id := e.confirmed(e.users[0], "A1", "A2")
	order, err := e.svc.Order(e.ctx, e.users[0], id)
	e.must(err)
	e.moveShowStart(e.showID, 10*time.Minute) // inside the check-in window
	if res, err := e.svc.Redeem(e.ctx, order.Tickets[0].Code, e.showID); err != nil || res.Status != models.RedeemOK {
		t.Fatalf("redeem: %+v %v", res, err)
	}
	e.mustHold(e.users[1], "B1")

	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	board, err := e.reports.StaffBoard(e.ctx, show.StartAt.UTC().Format(dto.DateLayout))
	e.must(err)
	var row *dto.StaffShowtimeResponse
	for i := range board.Showtimes {
		if board.Showtimes[i].ID == e.showID {
			row = &board.Showtimes[i]
		}
	}
	if row == nil {
		t.Fatalf("showtime missing from board %+v", board)
	}
	if row.Capacity != 9 || row.Sold != 2 || row.Held != 1 || row.CheckedIn != 1 || row.Available != 6 || row.HallName != "Hall 1" {
		t.Fatalf("board row = %+v", row)
	}

	waiting, err := e.reports.ShowtimeTickets(e.ctx, e.showID, models.TicketIssued)
	e.must(err)
	if len(waiting) != 1 || waiting[0].SeatLabel != order.Tickets[1].SeatLabel {
		t.Fatalf("waiting at the gate = %+v", waiting)
	}
	all, err := e.reports.ShowtimeTickets(e.ctx, e.showID, "")
	e.must(err)
	if len(all) != 2 {
		t.Fatalf("all tickets = %d", len(all))
	}
	if _, err := e.reports.ShowtimeTickets(e.ctx, e.showID, "lost"); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("bad status: err = %v", err)
	}
	if _, err := e.reports.ShowtimeTickets(e.ctx, "00000000-0000-0000-0000-000000000000", ""); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("unknown show: err = %v", err)
	}
	if _, err := e.reports.StaffBoard(e.ctx, "14/09/2026"); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("bad date: err = %v", err)
	}
}
