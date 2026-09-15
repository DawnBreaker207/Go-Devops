package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// T72a: the built-in templates preview a positive seat count and match what
// creating a hall from them actually produces.
func TestHallTemplates_PreviewMatchesCreatedGrid(t *testing.T) {
	e := newEnv(t)
	templates := e.halls.Templates()
	if len(templates) != 3 {
		t.Fatalf("templates = %d, want 3 (small, medium, large)", len(templates))
	}
	for _, tpl := range templates {
		if tpl.SeatCount <= 0 || tpl.Rows <= 0 || tpl.SeatsPerRow <= 0 {
			t.Fatalf("template %s = %+v", tpl.Name, tpl)
		}
		hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "From " + tpl.Name, Template: tpl.Name, Prices: fullPrices()})
		if err != nil {
			t.Fatalf("create from template %s: %v", tpl.Name, err)
		}
		seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
		e.must(err)
		got := 0
		for _, s := range seats {
			if !s.IsGap {
				got++
			}
		}
		if got != tpl.SeatCount {
			t.Fatalf("template %s: preview said %d seats, created hall has %d", tpl.Name, tpl.SeatCount, got)
		}
	}
}

// T72b: a request field explicitly set overrides the template's own value.
func TestHallTemplates_ExplicitFieldsOverride(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{
		Name: "Override", Template: "small", SeatsPerRow: 4, Prices: fullPrices(),
	})
	e.must(err)
	if hall.SeatsPerRow != 4 {
		t.Fatalf("seats_per_row = %d, want 4 (explicit override)", hall.SeatsPerRow)
	}
	if hall.Rows != 6 {
		t.Fatalf("rows = %d, want 6 (from the small template)", hall.Rows)
	}
}

// T73a: cloning copies the exact current grid (including a manual edit made
// after creation, not the original template) and, when asked, the prices.
func TestHallClone_CopiesGridAndPrices(t *testing.T) {
	e := newEnv(t)
	source, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Source", Rows: 2, SeatsPerRow: 3, Prices: fullPrices()})
	e.must(err)
	seats, err := e.halls.SeatsByHall(e.ctx, source.ID)
	e.must(err)
	var a1 string
	for _, s := range seats {
		if dto.SeatLabel(s.RowLabel, s.ColNumber) == "A1" {
			a1 = s.ID
		}
	}
	vip := models.SeatVIP
	_, err = e.halls.UpdateSeat(e.ctx, source.ID, a1, dto.SeatUpdateRequest{SeatType: vip})
	e.must(err)

	clone, err := e.halls.Clone(e.ctx, source.ID, dto.CloneHallRequest{Name: "Clone", CopyPrices: true})
	e.must(err)
	cloneSeats, err := e.halls.SeatsByHall(e.ctx, clone.ID)
	e.must(err)
	if len(cloneSeats) != len(seats) {
		t.Fatalf("clone has %d seats, want %d", len(cloneSeats), len(seats))
	}
	found := false
	for _, s := range cloneSeats {
		if dto.SeatLabel(s.RowLabel, s.ColNumber) == "A1" {
			found = true
			if s.SeatType != vip {
				t.Fatalf("clone A1 type = %s, want the manually-edited vip", s.SeatType)
			}
		}
	}
	if !found {
		t.Fatal("clone missing A1")
	}
	prices, err := e.halls.PricesByHall(e.ctx, clone.ID)
	e.must(err)
	if len(prices) != len(models.AllSeatTypes) {
		t.Fatalf("clone prices = %d, want %d", len(prices), len(models.AllSeatTypes))
	}

	uncopied, err := e.halls.Clone(e.ctx, source.ID, dto.CloneHallRequest{Name: "Clone 2", CopyPrices: false})
	e.must(err)
	if p, err := e.halls.PricesByHall(e.ctx, uncopied.ID); err != nil || len(p) != 0 {
		t.Fatalf("uncopied clone prices = %v, %v, want none", p, err)
	}
}

// T74: bulk seat edit covers all 4 selector kinds, is atomic, and is blocked
// by a live booking exactly like a single-seat edit.
func TestHallBulkUpdateSeats(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Bulk Hall", Rows: 4, SeatsPerRow: 4, Prices: fullPrices()})
	e.must(err)

	byLabel := func() map[string]models.Seat {
		seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
		e.must(err)
		m := map[string]models.Seat{}
		for _, s := range seats {
			m[dto.SeatLabel(s.RowLabel, s.ColNumber)] = s
		}
		return m
	}

	_, err = e.halls.BulkUpdateSeats(e.ctx, hall.ID, dto.BulkSeatUpdateRequest{Changes: []dto.SeatChange{
		{Selector: dto.SeatSelector{Labels: []string{"A1", "A2"}}, SeatType: models.SeatVIP},
		{Selector: dto.SeatSelector{Rows: []string{"B"}}, SeatType: models.SeatRecliner},
		{Selector: dto.SeatSelector{Cols: []int{4}}, IsGap: boolPtr(true)},
		{Selector: dto.SeatSelector{Range: "C1:C2"}, SeatType: models.SeatCouple},
	}})
	e.must(err)

	m := byLabel()
	if m["A1"].SeatType != models.SeatVIP || m["A2"].SeatType != models.SeatVIP {
		t.Fatalf("labels selector: A1=%s A2=%s", m["A1"].SeatType, m["A2"].SeatType)
	}
	if m["B1"].SeatType != models.SeatRecliner || m["B4"].SeatType != models.SeatRecliner {
		t.Fatalf("rows selector: B1=%s B4=%s", m["B1"].SeatType, m["B4"].SeatType)
	}
	if !m["A4"].IsGap || !m["D4"].IsGap {
		t.Fatalf("cols selector: A4.gap=%v D4.gap=%v", m["A4"].IsGap, m["D4"].IsGap)
	}
	if m["C1"].SeatType != models.SeatCouple || m["C2"].SeatType != models.SeatCouple || m["D1"].SeatType == models.SeatCouple {
		t.Fatalf("range selector: C1=%s C2=%s D1=%s", m["C1"].SeatType, m["C2"].SeatType, m["D1"].SeatType)
	}

	// A selector matching nothing, together with one that would match, rolls back atomically.
	before := byLabel()
	_, err = e.halls.BulkUpdateSeats(e.ctx, hall.ID, dto.BulkSeatUpdateRequest{Changes: []dto.SeatChange{
		{Selector: dto.SeatSelector{Labels: []string{"A3"}}, SeatType: models.SeatVIP},
		{Selector: dto.SeatSelector{Labels: []string{"Z9"}}, SeatType: models.SeatVIP},
	}})
	if !isAppErr(err, apperrors.ErrSeatValidation) {
		t.Fatalf("unmatched selector: err = %v", err)
	}
	if after := byLabel(); after["A3"].SeatType != before["A3"].SeatType {
		t.Fatal("a failed batch left a partial change")
	}

	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(5 * time.Hour)})
	e.must(err)
	seatMap, err := e.showtimes.SeatMap(e.ctx, show.ID)
	e.must(err)
	if _, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: show.ID, SeatIDs: []string{seatMap.Seats[0].ShowtimeSeatID}}); err != nil {
		t.Fatalf("hold on the bulk hall: %v", err)
	}
	if _, err := e.halls.BulkUpdateSeats(e.ctx, hall.ID, dto.BulkSeatUpdateRequest{
		Changes: []dto.SeatChange{{Selector: dto.SeatSelector{Labels: []string{"A2"}}, SeatType: models.SeatVIP}},
	}); !isAppErr(err, apperrors.ErrHallHasBookings) {
		t.Fatalf("bulk edit with a live hold: err = %v", err)
	}
}

func boolPtr(b bool) *bool { return &b }

// T75: deactivating a hall is refused while an open showtime is still to
// come; once inactive, no new showtime can be scheduled there.
func TestHallUpdate_DeactivateBlocksNewShowtimes(t *testing.T) {
	e := newEnv(t)
	name := "Deactivate Hall"
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: name, Rows: 1, SeatsPerRow: 2, Prices: fullPrices()})
	e.must(err)
	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(3 * time.Hour)})
	e.must(err)

	inactive := false
	if _, err := e.halls.UpdateHall(e.ctx, hall.ID, dto.UpdateHallRequest{Active: &inactive}); !isAppErr(err, apperrors.ErrHallStillSelling) {
		t.Fatalf("deactivate with an open upcoming showtime: err = %v", err)
	}

	e.must(e.showtimes.Delete(e.ctx, show.ID))
	updated, err := e.halls.UpdateHall(e.ctx, hall.ID, dto.UpdateHallRequest{Active: &inactive})
	e.must(err)
	if updated.Active {
		t.Fatal("hall still active")
	}
	if _, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(3 * time.Hour)}); !isAppErr(err, apperrors.ErrHallInactive) {
		t.Fatalf("create a showtime in an inactive hall: err = %v", err)
	}
}

// T76: the layout can only be regenerated before the hall ever had a booking,
// even one whose showtime was later deleted; regenerating rebuilds the grid.
func TestHallRegenerateLayout_BlockedAfterAnyBooking(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Layout Hall", Rows: 2, SeatsPerRow: 2, Prices: fullPrices()})
	e.must(err)

	regenerated, err := e.halls.RegenerateLayout(e.ctx, hall.ID, dto.HallRequest{Rows: 3, SeatsPerRow: 5, Prices: fullPrices()})
	e.must(err)
	if regenerated.Rows != 3 || regenerated.SeatsPerRow != 5 {
		t.Fatalf("regenerated hall = %+v", regenerated)
	}
	seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
	e.must(err)
	if len(seats) != 15 {
		t.Fatalf("seats after regenerate = %d, want 15", len(seats))
	}

	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(3 * time.Hour)})
	e.must(err)
	seatMap, err := e.showtimes.SeatMap(e.ctx, show.ID)
	e.must(err)
	held, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: show.ID, SeatIDs: []string{seatMap.Seats[0].ShowtimeSeatID}})
	e.must(err)
	_, err = e.svc.Cancel(e.ctx, e.users[0], held.BookingID)
	e.must(err)

	if _, err := e.halls.RegenerateLayout(e.ctx, hall.ID, dto.HallRequest{Rows: 1, SeatsPerRow: 1, Prices: fullPrices()}); !isAppErr(err, apperrors.ErrHallEverHadBookings) {
		t.Fatalf("regenerate after a canceled (but real) booking: err = %v", err)
	}
}

// T77: deleting a hall is refused while it has a showtime not yet ended.
func TestHallDelete_BlockedByUnfinishedShowtime(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Delete Hall", Rows: 1, SeatsPerRow: 2, Prices: fullPrices()})
	e.must(err)
	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(3 * time.Hour)})
	e.must(err)

	if err := e.halls.DeleteHall(e.ctx, hall.ID); !isAppErr(err, apperrors.ErrHallHasUpcomingShowtimes) {
		t.Fatalf("delete with a showtime to come: err = %v", err)
	}
	e.must(e.showtimes.Delete(e.ctx, show.ID))

	// A soft-deleted showtime is also forced closed, so nothing reading the
	// status column directly ever sees a "still open" deleted showtime.
	var deletedShow models.Showtime
	e.must(e.db.Unscoped().First(&deletedShow, "id = ?", show.ID).Error)
	if deletedShow.Status != models.ShowtimeClosed || !deletedShow.DeletedAt.Valid {
		t.Fatalf("deleted showtime = %+v, want status=closed and deleted_at set", deletedShow)
	}

	if err := e.halls.DeleteHall(e.ctx, hall.ID); err != nil {
		t.Fatalf("delete once its showtime is gone: %v", err)
	}
	if _, err := e.halls.GetByID(e.ctx, hall.ID); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("deleted hall still found: err = %v", err)
	}

	// A soft-deleted hall is also forced inactive, so active/deleted_at never
	// disagree forever.
	var deletedHall models.Hall
	e.must(e.db.Unscoped().First(&deletedHall, "id = ?", hall.ID).Error)
	if deletedHall.Active || !deletedHall.DeletedAt.Valid {
		t.Fatalf("deleted hall = %+v, want active=false and deleted_at set", deletedHall)
	}
}
