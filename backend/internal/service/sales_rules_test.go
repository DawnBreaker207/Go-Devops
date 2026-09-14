package service_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func TestShowtime_CleanupBufferBothSides(t *testing.T) {
	e := newEnv(t)
	var show models.Showtime // 100 minutes; the buffer is 20
	e.must(e.db.First(&show, "id = ?", e.showID).Error)
	create := func(start time.Time) error {
		_, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: start})
		return err
	}

	if err := create(show.StartAt.Add(-110 * time.Minute)); !isAppErr(err, apperrors.ErrShowtimeOverlap) {
		t.Fatalf("ending 10 minutes before the show: err = %v", err)
	}
	if err := create(show.EndAt.Add(10 * time.Minute)); !isAppErr(err, apperrors.ErrShowtimeOverlap) {
		t.Fatalf("starting 10 minutes after the show ended: err = %v", err)
	}
	if err := create(show.StartAt.Add(-120 * time.Minute)); err != nil {
		t.Fatalf("one buffer before the show: %v", err)
	}
	if err := create(show.EndAt.Add(20 * time.Minute)); err != nil {
		t.Fatalf("one buffer after the show: %v", err)
	}
}

// A movie no longer showing sells no seat and serves no seat map or realtime token.
func TestHold_MovieNotOnSale(t *testing.T) {
	e := newEnv(t)
	// Outside the API, which refuses to end a movie with showtimes to come.
	e.must(e.db.Exec(`UPDATE movies SET status = 'ended' WHERE id = ?`, e.movieID).Error)
	if _, err := e.hold(e.users[0], "A1"); !isAppErr(err, apperrors.ErrShowtimeClosed) {
		t.Fatalf("hold: err = %v", err)
	}
	if _, err := e.showtimes.SeatMap(e.ctx, e.showID); !isAppErr(err, apperrors.ErrShowtimeNotOpen) {
		t.Fatalf("seat map: err = %v", err)
	}
	if _, err := e.showtimes.OpenShowtime(e.ctx, e.showID); !isAppErr(err, apperrors.ErrShowtimeNotOpen) {
		t.Fatalf("realtime token: err = %v", err)
	}

	e.must(e.db.Exec(`UPDATE movies SET status = 'showing' WHERE id = ?`, e.movieID).Error)
	e.mustHold(e.users[0], "A1")
	e.checkInvariants()
}

func TestSeatMap_StartedShowNotServed(t *testing.T) {
	e := newEnv(t)
	e.moveShowStart(e.showID, -time.Minute)
	if _, err := e.showtimes.SeatMap(e.ctx, e.showID); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("seat map: err = %v, want 404", err)
	}
	if _, err := e.showtimes.OpenShowtime(e.ctx, e.showID); httpStatus(err) != http.StatusNotFound {
		t.Fatalf("realtime token: err = %v, want 404", err)
	}
}

func movieUpdate(status string, duration int) dto.MovieRequest {
	return dto.MovieRequest{Title: "Test Movie", Genre: "Drama", Duration: duration, Director: "Tester",
		ReleaseDate: "2026-09-01", Status: status}
}

// E-M2 / E-M4: closed showtimes allow ending the movie; only past ones free duration and deletion.
func TestMovies_DeleteAndStatusGuardedByUpcomingShowtimes(t *testing.T) {
	e := newEnv(t)
	update := func(status string, duration int) error {
		_, err := e.movies.Update(e.ctx, e.movieID, movieUpdate(status, duration))
		return err
	}

	if err := e.movies.Delete(e.ctx, e.movieID); !isAppErr(err, apperrors.ErrMovieHasShowtimes) {
		t.Fatalf("delete: err = %v", err)
	}
	if err := update(models.MovieStatusEnded, 100); !isAppErr(err, apperrors.ErrMovieHasShowtimes) {
		t.Fatalf("end: err = %v", err)
	}
	if err := update(models.MovieStatusShowing, 120); !isAppErr(err, apperrors.ErrMovieDurationLocked) {
		t.Fatalf("change duration: err = %v", err)
	}
	if err := update(models.MovieStatusShowing, 100); err != nil {
		t.Fatalf("edit the other fields: %v", err)
	}

	// A closed showtime could reopen with a wrong end, so the duration stays locked.
	e.must(e.db.Exec(`UPDATE showtimes SET status = 'closed' WHERE id = ?`, e.showID).Error)
	if err := update(models.MovieStatusShowing, 120); !isAppErr(err, apperrors.ErrMovieDurationLocked) {
		t.Fatalf("change duration with a closed showtime to come: err = %v", err)
	}
	if err := update(models.MovieStatusEnded, 100); err != nil {
		t.Fatalf("end once its showtimes are closed: %v", err)
	}

	e.moveShowStart(e.showID, -3*time.Hour)
	if err := update(models.MovieStatusEnded, 120); err != nil {
		t.Fatalf("change duration with past showtimes only: %v", err)
	}
	if err := e.movies.Delete(e.ctx, e.movieID); err != nil {
		t.Fatalf("delete with past showtimes only: %v", err)
	}
	if _, err := e.movies.GetByID(e.ctx, e.movieID, true); !isAppErr(err, apperrors.ErrMovieNotFound) {
		t.Fatalf("deleted movie still found: %v", err)
	}
}

// E-S3: the race never leaves an open showtime of an ended movie.
func TestRace_MovieEndVsShowtimeCreate(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 10; round++ {
		movie := &models.Movie{Title: fmt.Sprintf("Race %d", round), Genre: "Drama", Duration: 90,
			Director: "Tester", ReleaseDate: time.Now(), Status: models.MovieStatusShowing}
		e.must(e.db.Create(movie).Error)
		start := time.Now().Add(time.Duration(10+3*round) * time.Hour)

		var createErr, endErr error
		var wg sync.WaitGroup
		begin := make(chan struct{})
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			_, createErr = e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: movie.ID, HallID: e.hallID, StartAt: start})
		}()
		go func() {
			defer wg.Done()
			<-begin
			req := movieUpdate(models.MovieStatusEnded, 90)
			req.Title = movie.Title
			_, endErr = e.movies.Update(e.ctx, movie.ID, req)
		}()
		close(begin)
		wg.Wait()

		switch {
		case createErr == nil && endErr == nil:
			t.Fatalf("round %d: the showtime was scheduled and the movie ended", round)
		case createErr == nil && !isAppErr(endErr, apperrors.ErrMovieHasShowtimes):
			t.Fatalf("round %d: end after scheduling: err = %v", round, endErr)
		case endErr == nil && !isAppErr(createErr, apperrors.ErrMovieNotShowing):
			t.Fatalf("round %d: scheduling after the end: err = %v", round, createErr)
		}
	}
	if n := e.count(`SELECT COUNT(*) FROM showtimes s JOIN movies m ON m.id = s.movie_id
		WHERE s.status = 'open' AND s.start_at > NOW() AND m.status <> 'showing'`); n != 0 {
		t.Fatalf("%d open showtimes of movies not showing", n)
	}
}

// E-S7: closing always works; reopening needs a showing movie and a showtime still to come.
func TestShowtimeUpdate_CloseStartedShowOfEndedMovie(t *testing.T) {
	e := newEnv(t)
	e.confirmed(e.users[0], "A1")
	e.moveShowStart(e.showID, -10*time.Minute)
	e.must(e.db.Exec(`UPDATE movies SET status = 'ended' WHERE id = ?`, e.movieID).Error)

	var show models.Showtime
	set := func(status string) error {
		e.must(e.db.First(&show, "id = ?", e.showID).Error)
		_, err := e.showtimes.Update(e.ctx, e.showID, dto.ShowtimeRequest{
			MovieID: e.movieID, HallID: e.hallID, StartAt: show.StartAt, Status: status})
		return err
	}
	if err := set(models.ShowtimeClosed); err != nil {
		t.Fatalf("close a started showtime of an ended movie: %v", err)
	}
	if err := set(models.ShowtimeOpen); !isAppErr(err, apperrors.ErrShowtimeReopenLocked) {
		t.Fatalf("reopen for an ended movie: err = %v", err)
	}
	e.must(e.db.Exec(`UPDATE movies SET status = 'showing' WHERE id = ?`, e.movieID).Error)
	if err := set(models.ShowtimeOpen); !isAppErr(err, apperrors.ErrShowtimeReopenLocked) {
		t.Fatalf("reopen a started showtime: err = %v", err)
	}

	e.moveShowStart(e.showID, time.Hour)
	if err := set(models.ShowtimeOpen); err != nil {
		t.Fatalf("reopen a showtime still to come: %v", err)
	}
	e.checkInvariants()
}

// E-HO3: a retry returns the same hold; reuse for other seats or an ended booking answers 409.
func TestHold_IdempotencyKeyReuseRules(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]
	holdWith := func(key string, labels ...string) (*dto.HoldResponse, error) {
		return e.svc.Hold(e.ctx, u, dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids(labels...), IdempotencyKey: key})
	}

	first, err := holdWith("k-1", "A1", "A2")
	e.must(err)
	if again, err := holdWith("k-1", "A2", "A1"); err != nil || again.BookingID != first.BookingID {
		t.Fatalf("same request retried: %+v %v", again, err)
	}
	if _, err := holdWith("k-1", "A1", "A3"); !isAppErr(err, apperrors.ErrIdempotencyKeyReused) {
		t.Fatalf("key reused for another seat lot: err = %v", err)
	}

	e.payAndNotify(u, first.BookingID)
	e.wantStatus(first.BookingID, models.BookingConfirmed)
	if _, err := holdWith("k-1", "A1", "A2"); !isAppErr(err, apperrors.ErrIdempotencyKeyReused) {
		t.Fatalf("key of a confirmed booking: err = %v", err)
	}

	canceled, err := holdWith("k-2", "B1")
	e.must(err)
	_, err = e.svc.Cancel(e.ctx, u, canceled.BookingID)
	e.must(err)
	if _, err := holdWith("k-2", "B1"); !isAppErr(err, apperrors.ErrIdempotencyKeyReused) {
		t.Fatalf("key of a canceled hold: err = %v", err)
	}
	if _, err := holdWith("k-3", "B1"); err != nil {
		t.Fatalf("new key: %v", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM bookings WHERE idempotency_key IN ('k-1', 'k-2')`); n != 2 {
		t.Fatalf("bookings behind k-1/k-2 = %d, want 2", n)
	}
	e.checkInvariants()
}
