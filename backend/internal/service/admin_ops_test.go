package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// GET /admin/orders is the operator list: online and counter sales side by side,
// each carrying who bought it. A counter sale has no account, so its identity is
// the walk-in name and phone the till wrote down — a plain JOIN on users would
// drop it from the list entirely.
func TestHTTP_AdminOrderListShowsOnlineAndCounterSales(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)

	online := h.confirmed(h.users[0], "A1")

	status, _, body := h.call(http.MethodPost, "/api/v1/staff/orders", staff, map[string]any{
		"show_id": h.showID, "seat_ids": h.ids("A2"),
		"customer_name": "Walk In", "customer_phone": "0912345678",
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("counter sell: HTTP %d %v", status, body["message"])
	}
	counter := dataMap(body)["id"].(string)

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/orders", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("HTTP %d %v", status, body["message"])
	}
	rows := ordersByID(t, body)
	if len(rows) != 2 {
		t.Fatalf("items = %d, want 2 (one online, one counter)", len(rows))
	}

	onlineRow := rows[online]
	if onlineRow == nil {
		t.Fatalf("online order %s missing from %v", online, rows)
	}
	if onlineRow["sold_via"] != "online" {
		t.Errorf("online sold_via = %v", onlineRow["sold_via"])
	}
	if seats := int(onlineRow["seats"].(float64)); seats != 1 {
		t.Errorf("online seats = %d, want 1", seats)
	}
	customer, _ := onlineRow["customer"].(map[string]any)
	if customer["user_id"] != h.users[0] || customer["email"] != h.emailOf[h.users[0]] {
		t.Errorf("online customer = %v, want user %s / %s", customer, h.users[0], h.emailOf[h.users[0]])
	}
	payment, _ := onlineRow["payment"].(map[string]any)
	if payment == nil || payment["status"] != models.PaymentPaid {
		t.Errorf("online payment = %v, want a paid attempt", payment)
	}
	show, _ := onlineRow["showtime"].(map[string]any)
	if show["movie_title"] != "Test Movie" || show["hall_name"] != "Hall 1" {
		t.Errorf("online showtime = %v", show)
	}

	counterRow := rows[counter]
	if counterRow == nil {
		t.Fatalf("counter sale %s missing — a plain JOIN on users drops walk-ins", counter)
	}
	if counterRow["sold_via"] != "counter" {
		t.Errorf("counter sold_via = %v", counterRow["sold_via"])
	}
	if counterRow["payment"] != nil {
		t.Errorf("counter payment = %v, want none (cash carries no attempt)", counterRow["payment"])
	}
	customer, _ = counterRow["customer"].(map[string]any)
	if customer == nil || customer["full_name"] != "Walk In" || customer["phone"] != "0912345678" {
		t.Errorf("counter customer = %v, want the walk-in name and phone", customer)
	}
	if customer["user_id"] != nil {
		t.Errorf("counter customer.user_id = %v, want absent", customer["user_id"])
	}
}

// The filters narrow the same list without changing its shape, and an unknown
// enum value is a 400 with details keyed by the query parameter.
func TestHTTP_AdminOrderListFilters(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)

	online := h.confirmed(h.users[0], "A1")
	h.mustHold(h.users[1], "A3") // pending: only the status filter must surface it
	if status, _, body := h.call(http.MethodPost, "/api/v1/staff/orders", staff, map[string]any{
		"show_id": h.showID, "seat_ids": h.ids("A2"), "customer_name": "Walk In",
	}); status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("counter sell: HTTP %d %v", status, body["message"])
	}

	cases := []struct {
		query string
		want  int
	}{
		{"", 3},
		{"?status=confirmed", 2},
		{"?status=pending", 1},
		{"?status=refunded", 0},
		{"?sold_via=online", 2},
		{"?sold_via=counter", 1},
		{"?payment_status=paid", 1},
		{"?payment_status=failed", 0},
		{"?movie_id=" + h.movieID, 3},
		{"?showtime_id=" + h.showID, 3},
		{"?user_id=" + h.users[0], 1},
		{"?search=" + h.emailOf[h.users[0]], 1},
		{"?search=walk", 1},
		{"?search=" + online, 1},
		{"?search=nothing-matches-this", 0},
		{"?date=1999-01-01", 0},
	}
	for _, c := range cases {
		status, _, body := h.call(http.MethodGet, "/api/v1/admin/orders"+c.query, admin, nil)
		if status != http.StatusOK {
			t.Errorf("%q: HTTP %d %v", c.query, status, body["message"])
			continue
		}
		items, _ := dataMap(body)["items"].([]any)
		if len(items) != c.want {
			t.Errorf("%q: %d items, want %d", c.query, len(items), c.want)
		}
	}

	status, _, body := h.call(http.MethodGet, "/api/v1/admin/orders?status=bogus", admin, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("bad status enum: HTTP %d, want 400", status)
	}
	details, _ := body["details"].(map[string]any)
	if details["status"] == nil {
		t.Errorf("details = %v, want a message keyed by status", body["details"])
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/admin/orders?to=1999-01-01&from=2031-01-01", admin, nil); status != http.StatusBadRequest {
		t.Errorf("to before from: HTTP %d, want 400", status)
	}
}

// The four dashboard tiles. bookings counts CONFIRMED orders only — the bookings
// table is also the hold table, so a pending hold must not inflate it — and the
// three soft-deletable entities exclude deleted rows, which raw SQL only does
// because the predicate is written out by hand.
func TestHTTP_AdminStats(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	read := func() map[string]any {
		t.Helper()
		status, _, body := h.call(http.MethodGet, "/api/v1/admin/stats", admin, nil)
		if status != http.StatusOK {
			t.Fatalf("HTTP %d %v", status, body["message"])
		}
		return dataMap(body)
	}
	want := func(data map[string]any, key string, expected int64) {
		t.Helper()
		if got := int64(data[key].(float64)); got != expected {
			t.Errorf("%s = %d, want %d", key, got, expected)
		}
	}

	users := h.count(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	data := read()
	want(data, "movies", 1)
	want(data, "showtimes", 1)
	want(data, "bookings", 0)
	want(data, "users", users)

	h.mustHold(h.users[0], "A1") // pending hold: not a sale
	want(read(), "bookings", 0)

	h.confirmed(h.users[1], "A2")
	want(read(), "bookings", 1)

	// A soft-deleted movie leaves the row in place; the tile must not count it.
	extra, err := h.movies.Create(h.ctx, dto.MovieRequest{
		Title: "Deleted Later", Genre: "Drama", Duration: 90, Director: "Tester",
		ReleaseDate: "2026-01-01", Status: models.MovieStatusShowing,
	})
	h.must(err)
	want(read(), "movies", 2)
	h.must(h.movies.Delete(h.ctx, extra.ID))
	want(read(), "movies", 1)
}

// Both operator reads are admin-only: staff read one order at a time through
// /staff/orders/:id, never every customer's email next to every amount.
func TestHTTP_AdminOpsRoleScopes(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	for _, path := range []string{"/api/v1/admin/orders", "/api/v1/admin/stats"} {
		for _, c := range []struct {
			name, token string
			want        int
		}{
			{"admin", admin, http.StatusOK},
			{"staff", staff, http.StatusForbidden},
			{"customer", customer, http.StatusForbidden},
			{"anonymous", "", http.StatusUnauthorized},
		} {
			if status, _, body := h.call(http.MethodGet, path, c.token, nil); status != c.want {
				t.Errorf("%s as %s: HTTP %d (%v), want %d", path, c.name, status, body["message"], c.want)
			}
		}
	}
}

// ordersByID indexes an /admin/orders page by booking id.
func ordersByID(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()
	items, _ := dataMap(body)["items"].([]any)
	rows := make(map[string]map[string]any, len(items))
	for _, it := range items {
		row, _ := it.(map[string]any)
		id, _ := row["id"].(string)
		rows[id] = row
	}
	return rows
}
