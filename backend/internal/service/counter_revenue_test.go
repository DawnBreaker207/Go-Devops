package service_test

import (
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

// Counter day totals add tickets and combos; per-movie rows cover tickets only.
func TestReport_CounterDayAddsTicketsAndCombos(t *testing.T) {
	e := newEnv(t)
	today := time.Now().Format("2006-01-02")

	tix, err := e.svc.CounterSell(e.ctx, dto.CounterSellRequest{
		ShowID: e.showID, SeatIDs: e.ids("A1", "A2"),
	})
	if err != nil {
		t.Fatalf("counter ticket sell: %v", err)
	}
	_ = tix

	combos := service.NewComboService(e.db, repository.NewComboRepository(e.db),
		repository.NewComboOrderRepository(e.db), repository.NewBookingRepository(e.db))
	popcorn := seedCombo(t, e, "Bap quays", 59000)
	if _, err := combos.CounterSell(e.ctx, dto.CounterComboOrderRequest{
		Items:     []dto.ComboOrderItemRequest{{ComboID: popcorn, Quantity: 1}},
		PayMethod: "cash",
	}); err != nil {
		t.Fatalf("counter combo sell: %v", err)
	}

	day, err := e.reports.BoxOfficeDay(e.ctx, today)
	if err != nil {
		t.Fatalf("box office day: %v", err)
	}
	if day.Count != 1 || day.Total != 2*priceStandard {
		t.Fatalf("tickets = (%d, %d), want (1, %d)", day.Count, day.Total, 2*priceStandard)
	}
	if day.ComboCount != 1 || day.ComboTotal != 59000 {
		t.Fatalf("combos = (%d, %d), want (1, 59000)", day.ComboCount, day.ComboTotal)
	}

	movies, err := e.reports.CounterMovies(e.ctx, today)
	if err != nil {
		t.Fatalf("counter movies: %v", err)
	}
	if len(movies) != 1 || movies[0].Title != "Test Movie" {
		t.Fatalf("movies = %+v, want the one test movie", movies)
	}
	if movies[0].Tickets != 2 || movies[0].Revenue != 2*priceStandard {
		t.Fatalf("movie row = %+v", movies[0])
	}
}
