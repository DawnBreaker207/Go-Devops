package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// T72: a span anchor yields one 2-column seat and swallows its neighbor.
func TestHallLayout_SpanAnchorsConsumeNeighbor(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{
		Name: "Spa Hall", Rows: 2, SeatsPerRow: 5,
		SeatTypes: map[string][]string{"vip": {"2"}}, Gaps: []string{"A5"},
		Spans: []string{"A2"}, Prices: fullPrices(),
	})
	e.must(err)

	seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
	e.must(err)
	got := map[string]dto.SeatResponse{}
	for _, s := range seats {
		if s.ColSpan != 1 && s.ColSpan != 2 {
			t.Fatalf("seat %s has col_span %d", dto.SeatLabel(s.RowLabel, s.ColNumber), s.ColSpan)
		}
		got[dto.SeatLabel(s.RowLabel, s.ColNumber)] = dto.NewSeatResponse(&s)
	}
	if _, ok := got["A1"]; !ok {
		t.Fatal("A1 missing")
	}
	if s, ok := got["A2"]; !ok || s.ColSpan != 2 {
		t.Fatalf("A2 = %+v, want a spanning seat", got["A2"])
	}
	if _, ok := got["A3"]; ok {
		t.Fatal("A3 should be consumed by the A2 span")
	}
	if s := got["A5"]; !s.IsGap {
		t.Fatalf("A5 = %+v, want gap", got["A5"])
	}
	if len(seats) != 4 /* row A */ +5 /* row B */ {
		t.Fatalf("generated %d seats, want 9", len(seats))
	}
}

// T73: a span can not hang off the row edge, leave the grid, sit on a gap,
// repeat, or break the col_span check at the database.
func TestHallLayout_SpanValidation(t *testing.T) {
	e := newEnv(t)
	cases := []struct {
		name  string
		spans []string
		gaps  []string
	}{
		{"no room", []string{"B5"}, nil},               // B5 is the last column
		{"out of grid", []string{"B6"}, nil},           // column 6 > seats_per_row 5
		{"out of rows", []string{"C1"}, nil},           // rows = 2
		{"gap anchor", []string{"A5"}, []string{"A5"}}, // A5 is a gap
		{"duplicate anchor", []string{"A2"}, []string{"A2"}},
		// A2's consumed neighbor (A3) is itself already an anchor: naive
		// order-dependent validation missed this and silently dropped A4.
		{"overlapping spans, reverse order", []string{"A3", "A2"}, nil},
		{"overlapping spans, forward order", []string{"A2", "A3"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := e.halls.Create(e.ctx, dto.HallRequest{
				Name: "Bad Span Hall", Rows: 2, SeatsPerRow: 5,
				Gaps: c.gaps, Spans: c.spans, Prices: fullPrices()})
			if !isAppErr(err, apperrors.ErrSeatValidation) {
				t.Fatalf("err = %v, want seat validation", err)
			}
		})
	}
}

// T74: the seat map exposes the span and a spanning seat sells as one unit.
func TestSeatMap_SpanningSeatSellsAsOne(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{
		Name: "Love Hall", Rows: 1, SeatsPerRow: 4,
		Spans: []string{"A2"}, Prices: fullPrices(),
	})
	e.must(err)

	st, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(4 * time.Hour)})
	e.must(err)

	m, err := e.showtimes.SeatMap(e.ctx, st.ID)
	e.must(err)
	byLabel := map[string]dto.SeatMapSeat{}
	for _, s := range m.Seats {
		byLabel[s.Label] = s
	}
	if len(m.Seats) != 3 { // A1, A2 (span), A4; A3 consumed
		t.Fatalf("seat map has %d seats, want 3", len(m.Seats))
	}
	if byLabel["A2"].ColSpan != 2 || byLabel["A2"].Status != models.SeatStatusAvailable {
		t.Fatalf("A2 = %+v", byLabel["A2"])
	}
	if s, ok := byLabel["A3"]; ok {
		t.Fatalf("A3 leaked into the map: %+v", s)
	}

	seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
	e.must(err)
	var a2 string
	for _, s := range seats {
		if dto.SeatLabel(s.RowLabel, s.ColNumber) == "A2" {
			a2 = s.ID
		}
	}
	if a2 == "" {
		t.Fatal("hall has no A2 seat")
	}
	if _, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: st.ID, SeatIDs: []string{a2}}); err == nil {
		t.Fatal("physical seat id accepted, want a showtime-seat id")
	}
	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: st.ID, SeatIDs: []string{byLabel["A2"].ShowtimeSeatID}})
	e.must(err)
	if h.BookingID == "" {
		t.Fatal("no booking id")
	}
	b := e.payAndNotify(e.users[0], h.BookingID)
	if b.Status != models.BookingConfirmed || b.TotalAmount != priceStandard {
		t.Fatalf("couple-span sale = %+v", b)
	}
	if n := e.count(`SELECT COUNT(*) FROM tickets WHERE booking_id = ?`, h.BookingID); n != 1 {
		t.Fatalf("spanning seat produced %d tickets", n)
	}
}

// T75: the btree_gist exclusion rejects two overlapping showtimes in one hall,
// even when the app-level check is bypassed.
func TestShowtime_DatabaseRejectsPhantomOverlap(t *testing.T) {
	e := newEnv(t)
	start := time.Now().Add(20 * time.Hour).UTC()
	insert := func(from time.Time) error {
		return e.db.Exec(`INSERT INTO showtimes (id, movie_id, hall_id, start_at, end_at, status)
			VALUES (gen_random_uuid(), ?, ?, ?, ?, 'open')`,
			e.movieID, e.hallID, from, from.Add(100*time.Minute)).Error
	}
	if err := insert(start); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	err := insert(start.Add(30 * time.Minute)) // overlaps
	var pgErr *pgconn.PgError
	if err == nil || !errors.As(err, &pgErr) || pgErr.Code != "23P01" {
		t.Fatalf("overlapping insert err = %v, want exclusion violation 23P01", err)
	}
	// A second hall shares the load fine.
	if err := insert(start.Add(-5 * time.Hour)); err != nil {
		t.Fatalf("non-overlapping insert err = %v", err)
	}
}

// T76+T77: the partial exclusion ignores deleted rows, and adjacent windows
// with the cleanup buffer stay creatable through the API.
func TestShowtime_OverlapBackstopAndDeleteFrees(t *testing.T) {
	e := newEnv(t)

	first, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: time.Now().Add(12 * time.Hour)})
	e.must(err)
	if _, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: first.StartAt.Add(30 * time.Minute)}); !isAppErr(err, apperrors.ErrShowtimeOverlap) {
		t.Fatalf("app-level overlap: err = %v", err)
	}

	// The cleanup buffer (20min) leaves room right after the first show.
	tail := first.StartAt.Add(2*time.Hour + 20*time.Minute)
	if _, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: tail}); err != nil {
		t.Fatalf("adjacent showtime: %v", err)
	}

	e.must(e.showtimes.Delete(e.ctx, first.ID))
	reborn, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: first.StartAt})
	if err != nil {
		t.Fatalf("reuse the deleted window: %v", err)
	}
	if err := e.db.Exec(`UPDATE showtimes SET status = ? WHERE id = ?`, models.ShowtimeClosed, reborn.ID).Error; err != nil {
		t.Fatalf("close reborn: %v", err)
	}
}
