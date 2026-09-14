package service_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// Malformed ids are the client's fault (400), unknown referenced rows are 404.
func TestHTTP_BadIdsAnswer400Or404(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	for _, c := range []struct{ path, token string }{
		{"/api/v1/movies/not-a-uuid", ""},
		{"/api/v1/shows/not-a-uuid/seats", customer},
		{"/api/v1/orders/not-a-uuid", customer},
	} {
		if status, _, body := h.call(http.MethodGet, c.path, c.token, nil); status != http.StatusBadRequest {
			t.Errorf("%s: HTTP %d %v, want 400", c.path, status, body["message"])
		}
	}

	status, _, body := h.call(http.MethodPost, "/api/v1/admin/showtimes", admin, map[string]any{
		"movie_id": h.movieID, "hall_id": uuid.NewString(), "start_at": time.Now().Add(48 * time.Hour),
	})
	if status != http.StatusNotFound {
		t.Fatalf("showtime in an unknown hall: HTTP %d %v, want 404", status, body["message"])
	}
}

// Two registrations of one email at the same moment: one account, the other gets 409.
func TestRegister_ConcurrentSameEmail(t *testing.T) {
	e := newEnv(t)
	errs := make([]error, 6)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = e.auth.Register(e.ctx, dto.RegisterRequest{Email: "race@test.local", Password: "secret123", FullName: "Racer"})
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !isAppErr(err, apperrors.ErrEmailAlreadyExists):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || e.count(`SELECT COUNT(*) FROM users WHERE email = 'race@test.local'`) != 1 {
		t.Fatalf("registrations ok = %d, want exactly one account", ok)
	}
}

func TestHallSeatUpdate_PartialFieldsKeepGap(t *testing.T) {
	e := newEnv(t)
	var seatID string
	e.must(e.db.Raw(`SELECT id FROM seats WHERE hall_id = ? AND row_label = 'A' AND col_number = 5`, e.hallID).Scan(&seatID).Error)

	seat, err := e.halls.UpdateSeat(e.ctx, e.hallID, seatID, dto.SeatUpdateRequest{SeatType: models.SeatVIP})
	e.must(err)
	if !seat.IsGap || seat.SeatType != models.SeatVIP {
		t.Fatalf("type-only change: %+v, want a vip gap", seat)
	}
	open := false
	seat, err = e.halls.UpdateSeat(e.ctx, e.hallID, seatID, dto.SeatUpdateRequest{IsGap: &open})
	e.must(err)
	if seat.IsGap || seat.SeatType != models.SeatVIP {
		t.Fatalf("gap-only change: %+v, want a vip seat", seat)
	}
}

func TestHallCreate_AllStandardWithoutSeatTypes(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Plain Hall", Rows: 2, SeatsPerRow: 3, Prices: fullPrices()})
	e.must(err)
	if hall.SeatTypes == nil {
		t.Fatal("seat_types should be an empty object, not null")
	}
	if n := e.count(`SELECT COUNT(*) FROM seats WHERE hall_id = ? AND seat_type = 'standard'`, hall.ID); n != 6 {
		t.Fatalf("standard seats = %d, want 6", n)
	}
}

// Rows past Z (AA, AB) come after Z in seat grids and seat maps.
func TestHall_RowOrderBeyondZ(t *testing.T) {
	e := newEnv(t)
	hall, err := e.halls.Create(e.ctx, dto.HallRequest{Name: "Tall Hall", Rows: 28, SeatsPerRow: 1, Prices: fullPrices()})
	e.must(err)

	seats, err := e.halls.SeatsByHall(e.ctx, hall.ID)
	e.must(err)
	if len(seats) != 28 || seats[25].RowLabel != "Z" || seats[26].RowLabel != "AA" || seats[27].RowLabel != "AB" {
		t.Fatalf("grid order ends with %s, %s, %s", seats[25].RowLabel, seats[26].RowLabel, seats[27].RowLabel)
	}

	show, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(6 * time.Hour)})
	e.must(err)
	m, err := e.showtimes.SeatMap(e.ctx, show.ID)
	e.must(err)
	if got := m.Seats[len(m.Seats)-1].Label; got != "AB1" || m.Seats[0].Label != "A1" {
		t.Fatalf("seat map runs %s .. %s, want A1 .. AB1", m.Seats[0].Label, got)
	}
}

// An unknown email costs a bcrypt comparison too, so timing does not reveal registered emails.
func TestLogin_UnknownEmailCostsLikeWrongPassword(t *testing.T) {
	e := newEnv(t)
	_, err := e.auth.Register(e.ctx, dto.RegisterRequest{Email: "known@test.local", Password: "secret123", FullName: "Known"})
	e.must(err)

	fastest := func(email string) time.Duration {
		best := time.Hour
		for i := 0; i < 3; i++ {
			start := time.Now()
			// A new IP each time keeps the failed-login lockout out of the way.
			_, err := e.auth.Login(e.ctx, login(email, "wrong-pass", "10.7.0."+string(rune('1'+i))))
			if !isAppErr(err, apperrors.ErrInvalidCredentials) {
				t.Fatalf("login %s: err = %v", email, err)
			}
			best = min(best, time.Since(start))
		}
		return best
	}
	wrong, unknown := fastest("known@test.local"), fastest("nobody@test.local")
	t.Logf("wrong password %v, unknown email %v", wrong, unknown)
	if unknown < wrong/2 {
		t.Fatalf("unknown email answered in %v, wrong password in %v", unknown, wrong)
	}
}
