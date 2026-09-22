package service_test

import (
	"net/http"
	"testing"
)

// GET /pricing is the public price page. It must apply the SAME gate the customer
// showtime query does — active halls with all four seat types priced — or it will
// advertise a hall nobody can book.
func TestHTTP_PublicPricingListsOnlyBookableHalls(t *testing.T) {
	h := newHTTPEnv(t)

	// Anonymous: this is the whole point of the endpoint.
	status, _, body := h.call(http.MethodGet, "/api/v1/pricing", "", nil)
	if status != http.StatusOK {
		t.Fatalf("anonymous read: HTTP %d %v", status, body["message"])
	}
	data := dataMap(body)
	halls, _ := data["halls"].([]any)
	if len(halls) != 1 {
		t.Fatalf("halls = %d, want the 1 seeded hall", len(halls))
	}
	row, _ := halls[0].(map[string]any)
	prices, _ := row["prices"].(map[string]any)
	if len(prices) != 4 {
		t.Fatalf("prices = %v, want all four seat types", prices)
	}
	// from_price is the headline number, so it must be the cheapest seat anywhere.
	if got := num(t, data, "from_price"); got != priceStandard {
		t.Fatalf("from_price = %d, want the cheapest seat %d", got, priceStandard)
	}

	// A hall missing one price disappears, the same way it disappears from the
	// customer showtime list. Dropping one row is enough to fail the HAVING.
	h.must(h.db.Exec(`DELETE FROM hall_prices WHERE hall_id = ? AND seat_type = 'couple'`, h.hallID).Error)

	status, _, body = h.call(http.MethodGet, "/api/v1/pricing", "", nil)
	if status != http.StatusOK {
		t.Fatalf("after dropping a price: HTTP %d %v", status, body["message"])
	}
	data = dataMap(body)
	halls, _ = data["halls"].([]any)
	if len(halls) != 0 {
		t.Fatalf("halls = %d, want 0: a hall missing a seat-type price is not bookable", len(halls))
	}
	// An empty list must report 0, not a stale minimum.
	if got := num(t, data, "from_price"); got != 0 {
		t.Fatalf("from_price = %d on an empty list, want 0", got)
	}
}

// An inactive hall is rejected for new showtimes, so it has no business on a
// price page either.
func TestHTTP_PublicPricingSkipsInactiveHalls(t *testing.T) {
	h := newHTTPEnv(t)

	// Set the flag directly: UpdateHall refuses to deactivate a hall that still has
	// an open upcoming showtime, and that guard is not what this test is about -
	// the `active = TRUE` filter in PublicPriceList is.
	h.must(h.db.Exec(`UPDATE halls SET active = FALSE WHERE id = ?`, h.hallID).Error)

	status, _, body := h.call(http.MethodGet, "/api/v1/pricing", "", nil)
	if status != http.StatusOK {
		t.Fatalf("HTTP %d %v", status, body["message"])
	}
	if halls, _ := dataMap(body)["halls"].([]any); len(halls) != 0 {
		t.Fatalf("halls = %d, want an inactive hall left out", len(halls))
	}
}
