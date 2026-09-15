package service_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func together(n int, fn func(i int)) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

var oneSeatEach = [][]string{{"A1"}, {"A2"}, {"A3"}, {"A4"}, {"B1"}, {"B2"}, {"B3"}, {"B4"}}

// E-S4: either the delete or a hold wins; a deleted showtime never keeps a booking.
func TestRace_DeleteShowtimeVsHolds(t *testing.T) {
	deleted, refused := 0, 0
	for round := 0; round < 20; round++ {
		e := newEnv(t)
		holdErrs := make([]error, len(e.users))
		var delErr error
		together(len(e.users)+1, func(i int) {
			if i == len(e.users) {
				delErr = e.showtimes.Delete(e.ctx, e.showID)
				return
			}
			_, holdErrs[i] = e.hold(e.users[i], oneSeatEach[i]...)
		})

		switch {
		case delErr == nil:
			deleted++
		case isAppErr(delErr, apperrors.ErrShowtimeHasBookings):
			refused++
		default:
			t.Fatalf("round %d: delete: %v", round, delErr)
		}
		for i, err := range holdErrs {
			if err != nil && !isAppErr(err, apperrors.ErrShowtimeNotFound) {
				t.Errorf("round %d: hold %d: %v", round, i, err)
			}
		}
		if n := e.count(`SELECT COUNT(*) FROM bookings b JOIN showtimes s ON s.id = b.showtime_id
			WHERE s.deleted_at IS NOT NULL`); n != 0 {
			t.Errorf("round %d: showtime deleted while it has %d bookings", round, n)
		}
	}
	t.Logf("delete won %d rounds, refused %d rounds", deleted, refused)
}

// E-H4: either the gap change or a hold wins; a live booking never sits on a gap.
func TestRace_UpdateSeatVsHolds(t *testing.T) {
	changed, refused := 0, 0
	for round := 0; round < 20; round++ {
		e := newEnv(t)
		seats, err := e.halls.SeatsByHall(e.ctx, e.hallID)
		e.must(err)
		var a1 string
		for _, s := range seats {
			if dto.SeatLabel(s.RowLabel, s.ColNumber) == "A1" {
				a1 = s.ID
			}
		}

		holdErrs := make([]error, len(e.users))
		var updErr error
		together(len(e.users)+1, func(i int) {
			if i == len(e.users) {
				_, updErr = e.halls.UpdateSeat(e.ctx, e.hallID, a1, dto.SeatUpdateRequest{SeatType: models.SeatStandard, IsGap: new(true)})
				return
			}
			_, holdErrs[i] = e.hold(e.users[i], oneSeatEach[i]...)
		})

		switch {
		case updErr == nil:
			changed++
		case isAppErr(updErr, apperrors.ErrHallHasBookings):
			refused++
		default:
			t.Fatalf("round %d: update seat: %v", round, updErr)
		}
		for i, err := range holdErrs {
			if err != nil && !isAppErr(err, apperrors.ErrSeatNotSellable) {
				t.Errorf("round %d: hold %d: %v (HTTP %d)", round, i, err, httpStatus(err))
			}
		}
		if n := e.count(`SELECT COUNT(*) FROM booking_seats bs
			JOIN bookings b ON b.id = bs.booking_id
			JOIN showtime_seats ss ON ss.id = bs.showtime_seat_id
			JOIN seats s ON s.id = ss.seat_id
			WHERE s.is_gap AND b.status IN ('pending', 'confirmed')`); n != 0 {
			t.Errorf("round %d: %d live bookings sit on a gap", round, n)
		}
		if updErr == nil && holdErrs[0] == nil {
			t.Errorf("round %d: the layout changed and A1 was still held", round)
		}
	}
	t.Logf("layout change won %d rounds, refused %d rounds", changed, refused)
}

// E-S4: with live bookings only open/close is allowed; the hall changes only before any booking.
func TestShowtimeUpdate_ScheduleRules(t *testing.T) {
	e := newEnv(t)
	hall2, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Hall 2", Rows: 1, SeatsPerRow: 3,
		SeatTypes: map[string][]string{"vip": {"1"}}, Prices: fullPrices()})
	e.must(err)
	other := &models.Movie{Title: "Other Movie", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: time.Now(), Status: models.MovieStatusShowing}
	e.must(e.db.Create(other).Error)

	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	req := func(movieID, hallID string, start time.Time, status string) dto.ShowtimeRequest {
		return dto.ShowtimeRequest{MovieID: movieID, HallID: hallID, StartAt: start, Status: status}
	}
	grid := func(showID, hallID string) (all, inHall int64) {
		return e.count(`SELECT COUNT(*) FROM showtime_seats WHERE showtime_id = ?`, showID),
			e.count(`SELECT COUNT(*) FROM showtime_seats ss JOIN seats s ON s.id = ss.seat_id
				WHERE ss.showtime_id = ? AND s.hall_id = ?`, showID, hallID)
	}

	spare := e.newShowtime(8 * time.Hour)
	var spareRow models.Showtime
	e.must(e.db.First(&spareRow, "id = ?", spare).Error)
	_, err = e.showtimes.Update(e.ctx, spare, req(e.movieID, hall2.ID, spareRow.StartAt, ""))
	e.must(err)
	if all, in := grid(spare, hall2.ID); all != 3 || in != 3 {
		t.Fatalf("grid after moving to Hall 2: %d seats, %d of Hall 2", all, in)
	}
	if m, err := e.showtimes.SeatMap(e.ctx, spare); err != nil || len(m.Seats) != 3 {
		t.Fatalf("seat map after the move: %v", err)
	}

	h := e.mustHold(e.users[0], "A1")
	for name, r := range map[string]dto.ShowtimeRequest{
		"start": req(e.movieID, e.hallID, show.StartAt.Add(30*time.Minute), ""),
		"movie": req(other.ID, e.hallID, show.StartAt, ""),
	} {
		if _, err := e.showtimes.Update(e.ctx, e.showID, r); !isAppErr(err, apperrors.ErrShowtimeScheduleLocked) {
			t.Errorf("change %s with a pending booking: err = %v", name, err)
		}
	}
	if _, err := e.showtimes.Update(e.ctx, e.showID, req(e.movieID, hall2.ID, show.StartAt, "")); !isAppErr(err, apperrors.ErrShowtimeHallLocked) {
		t.Errorf("change hall with a pending booking: err = %v", err)
	}
	for _, status := range []string{models.ShowtimeClosed, models.ShowtimeOpen} {
		if _, err := e.showtimes.Update(e.ctx, e.showID, req(e.movieID, e.hallID, show.StartAt, status)); err != nil {
			t.Fatalf("set status %s with a pending booking: %v", status, err)
		}
	}

	_, err = e.svc.Cancel(e.ctx, e.users[0], h.BookingID)
	e.must(err)
	moved := show.StartAt.Add(30 * time.Minute)
	if _, err := e.showtimes.Update(e.ctx, e.showID, req(e.movieID, e.hallID, moved, "")); err != nil {
		t.Fatalf("move the time with only an expired booking: %v", err)
	}
	if _, err := e.showtimes.Update(e.ctx, e.showID, req(e.movieID, hall2.ID, moved, "")); !isAppErr(err, apperrors.ErrShowtimeHallLocked) {
		t.Fatalf("change hall with booking history: err = %v", err)
	}
	if all, in := grid(e.showID, e.hallID); all != 10 || in != 10 {
		t.Fatalf("grid of the booked showtime changed: %d seats, %d of Hall 1", all, in)
	}
}
