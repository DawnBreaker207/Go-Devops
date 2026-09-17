package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// T71: the age rating travels from the picker to the seat map, the order, the
// ticket email and the gate's redeem screen.
func TestAgeRating_ShowsEverywhere(t *testing.T) {
	e := newEnv(t)
	u := e.users[0]

	movie := &models.Movie{Title: "Rated Thirteen", Genre: "Drama", Duration: 95, Director: "Q",
		ReleaseDate: time.Now(), Status: models.MovieStatusShowing, AgeRating: "T13"}
	e.must(e.db.Create(movie).Error)
	start := time.Now().Add(8 * time.Hour)
	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: movie.ID, HallID: e.hallID, StartAt: start})
	e.must(err)
	showID := show.ID

	day := start.In(time.UTC).Format(dto.DateLayout)
	items, err := e.showtimes.ListByMovie(e.ctx, movie.ID, day)
	e.must(err)
	if len(items) != 1 || items[0].AgeRating != "T13" {
		t.Fatalf("picker items = %+v", items)
	}

	sm, err := e.showtimes.SeatMap(e.ctx, showID)
	e.must(err)
	if sm.AgeRating != "T13" {
		t.Fatalf("seat map age rating = %q", sm.AgeRating)
	}

	seats := e.seatsOf(showID)
	held, err := e.svc.Hold(e.ctx, u, dto.HoldRequest{ShowID: showID, SeatIDs: []string{seats["A1"]}})
	e.must(err)
	e.payAndNotify(u, held.BookingID)
	order, err := e.svc.Order(e.ctx, u, held.BookingID)
	e.must(err)
	if order.Showtime == nil || order.Showtime.AgeRating != "T13" {
		t.Fatalf("order showtime = %+v", order.Showtime)
	}

	if sent, err := e.emails.Send(e.ctx, held.BookingID); !sent || err != nil {
		t.Fatalf("email send: sent=%v err=%v", sent, err)
	}
	html := e.mailer.Sent()[0].HTML
	if !strings.Contains(html, "Độ tuổi") || !strings.Contains(html, "Từ 13 tuổi (T13)") {
		t.Fatalf("email misses the rating: %s", html)
	}

	e.moveShowStart(showID, 10*time.Minute)
	res, err := e.svc.Redeem(e.ctx, order.Tickets[0].Code, showID, "")
	e.must(err)
	if res.Status != models.RedeemOK || res.AgeRating != "T13" {
		t.Fatalf("redeem = %s / %q", res.Status, res.AgeRating)
	}
}
