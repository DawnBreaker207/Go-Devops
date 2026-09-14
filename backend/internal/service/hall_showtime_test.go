package service_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func hasShowtime(items []dto.ShowtimeListItem, id string) bool {
	return slices.ContainsFunc(items, func(it dto.ShowtimeListItem) bool { return it.ID == id })
}

// T28 / T29 / T30 (FR-CAT-01..03): showtimes of a day; empty day is an empty
// list; halls without full prices, ended movies and started shows are hidden.
func TestShowtimeListing_DayFilters(t *testing.T) {
	e := newEnv(t)
	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	day := show.StartAt.UTC().Format(dto.DateLayout)

	items, err := e.showtimes.ListByDate(e.ctx, day)
	e.must(err)
	if !hasShowtime(items, e.showID) {
		t.Fatalf("showtime missing from %s: %+v", day, items)
	}
	for _, it := range items {
		if it.ID == e.showID && (it.MovieTitle != "Test Movie" || it.FromPrice != priceStandard) {
			t.Fatalf("listing = %+v", it)
		}
	}

	// T28
	empty, err := e.showtimes.ListByDate(e.ctx, "2031-01-01")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty day = %#v, %v; want empty list", empty, err)
	}
	emptyMovie, err := e.showtimes.ListByMovie(e.ctx, e.movieID, "2031-01-01")
	if err != nil || emptyMovie == nil || len(emptyMovie) != 0 {
		t.Fatalf("empty movie day = %#v, %v", emptyMovie, err)
	}
	if _, err := e.showtimes.ListByDate(e.ctx, "01/01/2031"); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("bad date: err = %v", err)
	}

	// T30
	e.must(e.db.Exec(`DELETE FROM hall_prices WHERE hall_id = ? AND seat_type = 'couple'`, e.hallID).Error)
	if items, _ := e.showtimes.ListByDate(e.ctx, day); hasShowtime(items, e.showID) {
		t.Fatal("hall without full prices still listed")
	}
	if items, _ := e.showtimes.ListByMovie(e.ctx, e.movieID, day); hasShowtime(items, e.showID) {
		t.Fatal("hall without full prices still listed for the movie")
	}
	_, err = e.halls.SetPrices(e.ctx, e.hallID, dto.PriceRequest{Prices: fullPrices()})
	e.must(err)
	if items, _ := e.showtimes.ListByDate(e.ctx, day); !hasShowtime(items, e.showID) {
		t.Fatal("showtime not back after prices were restored")
	}

	// T29: ended movie
	e.must(e.db.Exec(`UPDATE movies SET status = 'ended' WHERE id = ?`, e.movieID).Error)
	if items, _ := e.showtimes.ListByDate(e.ctx, day); hasShowtime(items, e.showID) {
		t.Fatal("ended movie still listed")
	}
	e.must(e.db.Exec(`UPDATE movies SET status = 'showing' WHERE id = ?`, e.movieID).Error)

	// T29: already started
	e.must(e.db.Exec(`UPDATE showtimes SET start_at = NOW() - INTERVAL '1 minute' WHERE id = ?`, e.showID).Error)
	for _, d := range []string{day, time.Now().UTC().Format(dto.DateLayout)} {
		if items, _ := e.showtimes.ListByDate(e.ctx, d); hasShowtime(items, e.showID) {
			t.Fatalf("started showtime listed on %s", d)
		}
	}
}

// T31 / T32 (FR-HALL-01/02): the grid follows rows, seat types and gaps; a
// hall is never created or saved without a positive price for every type.
func TestHallCreate_GridAndPrices(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{
		Name: "Grid Hall", Rows: 4, SeatsPerRow: 6,
		SeatTypes: map[string][]string{"vip": {"2"}, "couple": {"3"}, "recliner": {"4"}},
		Gaps:      []string{"A1", "D6"},
		Prices:    fullPrices(),
	})
	e.must(err)
	seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
	e.must(err)
	if len(seats) != 24 {
		t.Fatalf("seats = %d, want 24", len(seats))
	}
	byType := map[string]int{}
	var gaps []string
	for _, s := range seats {
		byType[s.SeatType]++
		if s.IsGap {
			gaps = append(gaps, dto.SeatLabel(s.RowLabel, s.ColNumber))
		}
	}
	for seatType, want := range map[string]int{"standard": 6, "vip": 6, "couple": 6, "recliner": 6} {
		if byType[seatType] != want {
			t.Fatalf("seat types = %v", byType)
		}
	}
	slices.Sort(gaps)
	if !slices.Equal(gaps, []string{"A1", "D6"}) {
		t.Fatalf("gaps = %v", gaps)
	}
	if prices, _ := e.halls.PricesByHall(e.ctx, hall.ID); len(prices) != 4 {
		t.Fatalf("prices = %d, want 4", len(prices))
	}

	missing := fullPrices()
	delete(missing, models.SeatRecliner)
	if _, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "No Price Hall", Rows: 2, SeatsPerRow: 2,
		SeatTypes: map[string][]string{"vip": {"2"}}, Prices: missing}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("hall without every price: err = %v", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM halls WHERE name = 'No Price Hall'`); n != 0 {
		t.Fatal("hall created without every price")
	}
	zero := fullPrices()
	zero[models.SeatVIP] = 0
	if _, err := e.halls.SetPrices(e.ctx, hall.ID, dto.PriceRequest{Prices: zero}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("zero price: err = %v", err)
	}
	if _, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Grid Hall", Rows: 1, SeatsPerRow: 1,
		SeatTypes: map[string][]string{"vip": {"1"}}, Prices: fullPrices()}); httpStatus(err) != http.StatusConflict {
		t.Fatalf("duplicate name: err = %v", err)
	}
	if _, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Bad Gap", Rows: 2, SeatsPerRow: 2,
		SeatTypes: map[string][]string{"vip": {"2"}}, Gaps: []string{"Z9"}, Prices: fullPrices()}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("gap outside the grid: err = %v", err)
	}
}

// T33 (FR-HALL-03): a live booking (pending/confirmed) locks the layout with
// 409; an expired booking does not.
func TestHallLayout_LockedOnlyByLiveBookings(t *testing.T) {
	e := newEnv(t)
	seats, err := e.halls.SeatsByHall(e.ctx, e.hallID)
	e.must(err)
	var b5 string
	for _, s := range seats {
		if dto.SeatLabel(s.RowLabel, s.ColNumber) == "B5" {
			b5 = s.ID
		}
	}

	h := e.mustHold(e.users[0], "A1")
	if _, err := e.halls.UpdateSeat(e.ctx, e.hallID, b5, dto.SeatUpdateRequest{SeatType: models.SeatCouple}); httpStatus(err) != http.StatusConflict {
		t.Fatalf("pending booking: err = %v, want 409", err)
	}
	_, err = e.svc.Cancel(e.ctx, e.users[0], h.BookingID)
	e.must(err)
	if _, err := e.halls.UpdateSeat(e.ctx, e.hallID, b5, dto.SeatUpdateRequest{SeatType: models.SeatCouple}); err != nil {
		t.Fatalf("expired booking still locks the layout: %v", err)
	}
	e.confirmed(e.users[1], "A2")
	if _, err := e.halls.UpdateSeat(e.ctx, e.hallID, b5, dto.SeatUpdateRequest{SeatType: models.SeatVIP}); httpStatus(err) != http.StatusConflict {
		t.Fatalf("confirmed booking: err = %v, want 409", err)
	}
}

// T34 (FR-SHOW-02): a showtime with bookings can not be deleted (409); closing
// it stops sales.
func TestShowtime_DeleteAndClose(t *testing.T) {
	e := newEnv(t)
	spare := e.newShowtime(8 * time.Hour)
	if err := e.showtimes.Delete(e.ctx, spare); err != nil {
		t.Fatalf("delete an empty showtime: %v", err)
	}
	e.mustHold(e.users[0], "A1")
	if err := e.showtimes.Delete(e.ctx, e.showID); httpStatus(err) != http.StatusConflict {
		t.Fatalf("delete with bookings: err = %v, want 409", err)
	}

	var show models.Showtime
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	_, err := e.showtimes.Update(e.ctx, e.showID, dto.ShowtimeRequest{
		MovieID: e.movieID, HallID: e.hallID, StartAt: show.StartAt, Status: models.ShowtimeClosed,
	})
	e.must(err)
	if _, err := e.hold(e.users[1], "A2"); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("hold on a closed show: err = %v", err)
	}
	if _, err := e.showtimes.SeatMap(e.ctx, e.showID); err == nil {
		t.Fatal("closed show still served a seat map")
	}
}

// T35 (FR-SEAT-01): the seat map shows each seat's state and its type price.
func TestSeatMap_StatusesAndPrices(t *testing.T) {
	e := newEnv(t)
	e.confirmed(e.users[0], "A1")
	e.mustHold(e.users[1], "B1")

	m, err := e.showtimes.SeatMap(e.ctx, e.showID)
	e.must(err)
	if len(m.Seats) != 10 || m.Prices[models.SeatVIP] != priceVIP || m.Prices[models.SeatStandard] != priceStandard {
		t.Fatalf("seat map: %d seats, prices %v", len(m.Seats), m.Prices)
	}
	byLabel := map[string]dto.SeatMapSeat{}
	for _, s := range m.Seats {
		byLabel[s.Label] = s
	}
	checks := []struct {
		label, status string
		price         int64
	}{
		{"A1", models.SeatStatusSold, priceStandard},
		{"B1", models.SeatStatusHeld, priceVIP},
		{"A2", models.SeatStatusAvailable, priceStandard},
		{"B2", models.SeatStatusAvailable, priceVIP},
	}
	for _, c := range checks {
		if s := byLabel[c.label]; s.Status != c.status || s.Price != c.price {
			t.Fatalf("%s = %+v, want %s at %d", c.label, s, c.status, c.price)
		}
	}
	if !byLabel["A5"].IsGap {
		t.Fatal("A5 should be a gap")
	}
}
