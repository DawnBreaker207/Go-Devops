package service_test

import (
	"net/http"
	"testing"
)

// GET /pricing is the public price page. Phase 3 of the pricing redesign
// (PLAN_CAMPAIGN.md section 11.4) dropped hall_prices, so this is no longer
// a per-hall listing: every hall shares the one global seat_base_prices row
// per seat type, and that is exactly what the page now echoes.
func TestHTTP_PublicPricingListsGlobalBasePrices(t *testing.T) {
	h := newHTTPEnv(t)

	// Anonymous: this is the whole point of the endpoint.
	status, _, body := h.call(http.MethodGet, "/api/v1/pricing", "", nil)
	if status != http.StatusOK {
		t.Fatalf("anonymous read: HTTP %d %v", status, body["message"])
	}
	data := dataMap(body)
	prices, _ := data["prices"].(map[string]any)
	if len(prices) != 4 {
		t.Fatalf("prices = %v, want all four seat types", prices)
	}
	// from_price is the headline number, so it must be the cheapest configured seat.
	if got := num(t, data, "from_price"); got != priceStandard {
		t.Fatalf("from_price = %d, want the cheapest seat %d", got, priceStandard)
	}

	// Dropping every seat type's base price back to 0 (nothing configured)
	// must report from_price 0, not a stale minimum.
	h.must(h.db.Exec(`UPDATE seat_base_prices SET price = 0`).Error)

	status, _, body = h.call(http.MethodGet, "/api/v1/pricing", "", nil)
	if status != http.StatusOK {
		t.Fatalf("after zeroing prices: HTTP %d %v", status, body["message"])
	}
	data = dataMap(body)
	if got := num(t, data, "from_price"); got != 0 {
		t.Fatalf("from_price = %d with nothing configured, want 0", got)
	}
	prices, _ = data["prices"].(map[string]any)
	if len(prices) != 4 {
		t.Fatalf("prices = %v, want all four seat types still listed (at 0)", prices)
	}
}
