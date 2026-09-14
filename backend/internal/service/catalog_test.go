package service_test

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// T9 / E-S5: two overlapping showtimes created at once in one hall, exactly one wins.
func TestShowtime_ConcurrentOverlapOneWins(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		base := time.Now().Add(time.Duration(10+round*5) * time.Hour)
		var ok, conflict atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{
					MovieID: e.movieID, HallID: e.hallID, StartAt: base.Add(time.Duration(i*10) * time.Minute),
				})
				switch {
				case err == nil:
					ok.Add(1)
				case isAppErr(err, apperrors.ErrShowtimeOverlap):
					conflict.Add(1)
				default:
					t.Errorf("round %d: %v", round, err)
				}
			}(i)
		}
		wg.Wait()
		if ok.Load() != 1 || conflict.Load() != 1 {
			t.Fatalf("round %d: ok=%d conflict=%d, want 1/1", round, ok.Load(), conflict.Load())
		}
	}
}

// E-CAT2: also the status filter and an empty showtime list.
func TestMovies_DraftsHiddenFromCustomers(t *testing.T) {
	e := newEnv(t)
	draft := &models.Movie{Title: "Secret Cut", Genre: "Drama", Duration: 90, Director: "X", ReleaseDate: time.Now(), Status: models.MovieStatusDraft}
	ended := &models.Movie{Title: "Old Hit", Genre: "Drama", Duration: 90, Director: "X", ReleaseDate: time.Now(), Status: models.MovieStatusEnded}
	e.must(e.db.Create(draft).Error)
	e.must(e.db.Create(ended).Error)
	page := dto.PageQuery{Page: 1, PageSize: 10}

	if _, total, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page}, true); err != nil || total != 3 {
		t.Fatalf("admin list total = %d, %v", total, err)
	}
	items, total, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page}, false)
	e.must(err)
	if total != 2 {
		t.Fatalf("customer list total = %d, want 2", total)
	}
	for _, m := range items {
		if m.Status == models.MovieStatusDraft {
			t.Fatal("customer sees a draft")
		}
	}
	if _, total, _ := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Status: models.MovieStatusShowing}, false); total != 1 {
		t.Fatalf("showing filter total = %d, want 1", total)
	}
	if _, total, _ := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Status: models.MovieStatusDraft}, false); total != 0 {
		t.Fatalf("customer draft filter total = %d, want 0", total)
	}
	if _, err := e.movies.GetByID(e.ctx, draft.ID, false); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("customer draft detail: err = %v", err)
	}
	if _, err := e.movies.GetByID(e.ctx, draft.ID, true); err != nil {
		t.Fatalf("admin draft detail: %v", err)
	}
	shows, err := e.showtimes.ListByMovie(e.ctx, ended.ID, "")
	if err != nil || len(shows) != 0 {
		t.Fatalf("ended movie showtimes = %v, %v; want empty list", shows, err)
	}
}

// T64: C10 shows the extra metadata, respects the genre filter, and sorts safely.
func TestMovies_ExtraFieldsFilterAndSort(t *testing.T) {
	e := newEnv(t)
	page := dto.PageQuery{Page: 1, PageSize: 20}

	made, err := e.movies.Create(e.ctx, dto.MovieRequest{
		Title: "Rated Film", Genre: "Sci-Fi", Duration: 110, Director: "Y",
		PosterURL: "http://poster.test/p.png", TrailerURL: "https://www.youtube.com/watch?v=uniquetoken",
		Cast: "Nguyen Van A, Tran Thi B", AgeRating: "T16",
		ReleaseDate: time.Now().Format(dto.DateLayout), Status: models.MovieStatusShowing,
	})
	e.must(err)
	if made.TrailerURL == "" || made.Cast != "Nguyen Van A, Tran Thi B" || made.AgeRating != "T16" {
		t.Fatalf("created = %+v", made)
	}
	blank, err := e.movies.Create(e.ctx, dto.MovieRequest{
		Title: "Solid", Genre: "Drama", Duration: 90, Director: "Z",
		ReleaseDate: time.Now().Format(dto.DateLayout), Status: models.MovieStatusShowing,
	})
	e.must(err)
	if blank.AgeRating != "P" || blank.TrailerURL != "" || blank.Cast != "" {
		t.Fatalf("defaults = %+v", blank)
	}

	e.must(e.db.Create(&models.Movie{Title: "Zulu Cut", Genre: "Action", Duration: 90, Director: "X", ReleaseDate: time.Now().AddDate(0, 0, -1), Status: models.MovieStatusShowing}).Error)
	e.must(e.db.Create(&models.Movie{Title: "Aces", Genre: "Action", Duration: 90, Director: "X", ReleaseDate: time.Now().AddDate(0, 0, -2), Status: models.MovieStatusShowing}).Error)

	items, total, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Genre: "action"}, false)
	e.must(err)
	if total != 2 {
		t.Fatalf("genre total = %d, want 2", total)
	}
	if items[0].Title != "Aces" || items[1].Title != "Zulu Cut" {
		t.Fatalf("genre list not title-asc: %+v", items)
	}
	byTitle, _, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Sort: "title", Order: "desc"}, false)
	e.must(err)
	if byTitle[0].Title != "Zulu Cut" {
		t.Fatalf("title desc first = %s", byTitle[0].Title)
	}
	byRelease, _, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Sort: "release_date", Order: "asc"}, false)
	e.must(err)
	if byRelease[0].Title != "Aces" {
		t.Fatalf("release asc first = %s", byRelease[0].Title)
	}
	if _, total, err := e.movies.List(e.ctx, dto.MovieListQuery{PageQuery: page, Sort: "wall-clock", Order: "sideways"}, false); err != nil || total != 5 {
		t.Fatalf("bogus sort falls back, total = %d, err = %v", total, err)
	}
}

func TestCancelHold_ReleasesSeatsImmediately(t *testing.T) {
	e := newEnv(t)
	sub, err := e.hub.Subscribe(e.showID, "viewer")
	e.must(err)
	defer sub.Close()
	u := e.users[0]

	h := e.mustHold(u, "A1", "A2")
	expectEvent(t, sub.Events, models.SeatStatusHeld, 2)
	if _, err := e.svc.Cancel(e.ctx, e.users[1], h.BookingID); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("someone else cancels: err = %v", err)
	}

	st, err := e.svc.Cancel(e.ctx, u, h.BookingID)
	e.must(err)
	if st.Status != models.BookingExpired || st.StatusReason != models.ReasonCanceled {
		t.Fatalf("status = %s/%s", st.Status, st.StatusReason)
	}
	expectEvent(t, sub.Events, models.SeatStatusAvailable, 2)
	e.wantSeat("A1", models.SeatStatusAvailable)
	e.wantSeat("A2", models.SeatStatusAvailable)
	if _, err := e.svc.Cancel(e.ctx, u, h.BookingID); !isAppErr(err, apperrors.ErrBookingExpired) {
		t.Fatalf("cancel twice: err = %v", err)
	}

	id := e.confirmed(u, "B1")
	if _, err := e.svc.Cancel(e.ctx, u, id); !isAppErr(err, apperrors.ErrBookingNotPending) {
		t.Fatalf("cancel a confirmed booking: err = %v", err)
	}
	e.checkInvariants()
}
