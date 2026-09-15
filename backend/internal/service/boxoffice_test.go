package service_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func TestBoxOffice_WalkInSellsConfirmedNoEmail(t *testing.T) {
	e := newEnv(t)

	res, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{
		ShowID: e.showID, SeatIDs: e.ids("A1", "A2"),
		CustomerName: "Walk-in Guest", CustomerPhone: "0900000000",
	})
	if err != nil {
		t.Fatalf("counter sell: %v", err)
	}
	if res.ID == "" || res.Status != models.BookingConfirmed || res.Payment != nil {
		t.Fatalf("order = %+v", res)
	}
	if res.TotalAmount != 2*priceStandard {
		t.Fatalf("total = %d, want %d", res.TotalAmount, 2*priceStandard)
	}
	if len(res.Tickets) != 2 || slices.ContainsFunc(res.Tickets, func(tr dto.TicketResponse) bool { return tr.Code == "" || tr.Status != models.TicketIssued }) {
		t.Fatalf("tickets = %+v", res.Tickets)
	}

	b := e.booking(res.ID)
	if b.Status != models.BookingConfirmed || b.SoldVia != models.SoldViaCounter {
		t.Fatalf("booking = %+v", b)
	}
	if b.UserID != "" || b.CustomerName != "Walk-in Guest" || b.CustomerPhone != "0900000000" {
		t.Fatalf("walk-in snapshot not stored: %+v", b)
	}
	if b.PaidAt == nil {
		t.Fatal("counter sale was not marked paid")
	}
	e.wantSeat("A1", models.SeatStatusSold)
	e.wantSeat("A2", models.SeatStatusSold)

	ids, err := e.emails.PendingIDs(e.ctx, 50)
	e.must(err)
	if slices.Contains(ids, res.ID) {
		t.Fatalf("counter booking %s is queued for a ticket email", res.ID)
	}
	if sent, err := e.emails.Send(e.ctx, res.ID); sent || err != nil {
		t.Fatalf("send for counter booking: sent=%v err=%v", sent, err)
	}
}

func TestBoxOffice_RefusesUnavailableInputs(t *testing.T) {
	e := newEnv(t)

	held := e.mustHold(e.users[0], "A1")
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("A1", "A2")}); !isAppErr(err, apperrors.ErrSeatTaken) {
		t.Fatalf("seat held online: err = %v", err)
	}
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("A5")}); !isAppErr(err, apperrors.ErrSeatNotSellable) {
		t.Fatalf("gap seat: err = %v", err)
	}
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("A1", "A1")}); err == nil {
		t.Fatal("duplicate seats accepted")
	}
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("A1", "A2", "B1", "B2", "B3")}); !isAppErr(err, apperrors.ErrSeatLimitExceeded) {
		t.Fatalf("over the seat cap: err = %v", err)
	}

	started := e.newShowtime(5 * time.Hour)
	e.moveShowStart(started, -30*time.Minute)
	stSeats := e.seatsOf(started)
	ids := []string{stSeats["A1"], stSeats["A2"]}
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: started, SeatIDs: ids}); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("started showtime: err = %v", err)
	}

	closed := e.newShowtime(7 * time.Hour)
	e.must(e.db.Exec(`UPDATE showtimes SET status = ? WHERE id = ?`, models.ShowtimeClosed, closed).Error)
	clSeats := e.seatsOf(closed)
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: closed, SeatIDs: []string{clSeats["A1"]}}); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("closed showtime: err = %v", err)
	}

	// Nothing was sold: the online hold is intact and no counter booking leaked.
	if b := e.booking(held.BookingID); b.Status != models.BookingPending {
		t.Fatalf("held booking status = %s", b.Status)
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE sold_via = 'counter'`); n != 0 {
		t.Fatalf("counter bookings after refusals = %d", n)
	}
}

func TestBoxOffice_OnlineHoldAndCounterSellCoexist(t *testing.T) {
	e := newEnv(t)

	held := e.mustHold(e.users[0], "A1", "A2")
	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("B1", "B2")}); err != nil {
		t.Fatalf("counter sell while seats held elsewhere: %v", err)
	}
	e.wantSeat("A1", models.SeatStatusHeld)
	e.wantSeat("B1", models.SeatStatusSold)

	online := e.payAndNotify(e.users[0], held.BookingID)
	if online.Status != models.BookingConfirmed {
		t.Fatalf("online booking = %+v", online)
	}
	e.wantSeat("A1", models.SeatStatusSold)
	if b := e.booking(held.BookingID); b.SoldVia != models.SoldViaOnline {
		t.Fatalf("online booking marked %s", b.SoldVia)
	}
	ids, err := e.emails.PendingIDs(e.ctx, 50)
	e.must(err)
	if !slices.Contains(ids, held.BookingID) {
		t.Fatalf("online booking missing from email queue: %v", ids)
	}
}

func TestBoxOffice_DayReportCountsCounterOnly(t *testing.T) {
	e := newEnv(t)

	if _, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{ShowID: e.showID, SeatIDs: e.ids("A1", "A2")}); err != nil {
		t.Fatalf("counter sell: %v", err)
	}
	e.confirmed(e.users[0], "B1", "B2")

	day, err := e.reports.BoxOfficeDay(e.ctx, "")
	e.must(err)
	wantDate := time.Now().In(time.UTC).Format(dto.DateLayout)
	if day.Date != wantDate || day.Count != 1 || day.Total != 2*priceStandard {
		t.Fatalf("day report = %+v, want date %s count 1 total %d", day, wantDate, 2*priceStandard)
	}

	// The date form is parsed back to the same window, including a bad one.
	if _, err := e.reports.BoxOfficeDay(e.ctx, "not-a-date"); err == nil {
		t.Fatal("bad date accepted")
	}
}
