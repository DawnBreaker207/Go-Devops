package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// createCode adds a discount code through the admin API and returns its id.
func createCode(t *testing.T, h *httpEnv, admin string, payload map[string]any) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/discounts", admin, payload)
	if status != http.StatusCreated {
		t.Fatalf("create discount: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["id"].(string)
	return id
}

// holdSeats opens a pending order for a customer and returns its booking id.
func holdSeats(t *testing.T, h *httpEnv, token string, labels ...string) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/orders/hold", token, map[string]any{
		"show_id": h.showID, "seat_ids": h.ids(labels...),
	})
	if status != http.StatusOK && status != http.StatusCreated {
		t.Fatalf("hold: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["booking_id"].(string)
	if id == "" {
		t.Fatalf("hold returned no booking_id: %v", body)
	}
	return id
}

func num(t *testing.T, m map[string]any, key string) int64 {
	t.Helper()
	v, ok := m[key].(float64)
	if !ok {
		t.Fatalf("field %q missing or not a number in %v", key, m)
	}
	return int64(v)
}

// THE central invariant of this feature. bookings.total_amount must stay the
// undiscounted seat subtotal, because finalizeTx asserts the sold seats add up
// to it — a discounted total_amount would make every discounted order fail to
// confirm AFTER the customer's money had already been taken. The discount must
// instead reach the gateway through payments.amount.
func TestHTTP_DiscountChargesLessButKeepsTheSeatSubtotal(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{
		"code": "WELCOME10", "kind": "percent", "value": 10,
	})

	bookingID := holdSeats(t, h, customer, "A1", "A2")

	status, _, body := h.call(http.MethodGet, "/api/v1/orders/"+bookingID, customer, nil)
	if status != http.StatusOK {
		t.Fatalf("read order: HTTP %d %v", status, body["message"])
	}
	subtotal := num(t, dataMap(body), "total_amount")
	if subtotal <= 0 {
		t.Fatalf("subtotal = %d, want the two seats' prices", subtotal)
	}

	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "welcome10"}) // lower case on purpose: codes match uppercase
	if status != http.StatusOK {
		t.Fatalf("apply: HTTP %d %v", status, body["message"])
	}
	applied := dataMap(body)
	want := subtotal / 10
	if got := num(t, applied, "discount"); got != want {
		t.Fatalf("discount = %d, want %d (10%% of %d)", got, want, subtotal)
	}
	if got := num(t, applied, "payable"); got != subtotal-want {
		t.Fatalf("payable = %d, want %d", got, subtotal-want)
	}
	if got := num(t, applied, "subtotal"); got != subtotal {
		t.Fatalf("subtotal changed to %d, want it untouched at %d", got, subtotal)
	}

	// Re-read rather than trusting the echo.
	status, _, body = h.call(http.MethodGet, "/api/v1/orders/"+bookingID, customer, nil)
	if status != http.StatusOK {
		t.Fatalf("re-read: HTTP %d %v", status, body["message"])
	}
	order := dataMap(body)
	if got := num(t, order, "total_amount"); got != subtotal {
		t.Fatalf("total_amount = %d, want the UNDISCOUNTED subtotal %d", got, subtotal)
	}
	if got := num(t, order, "payable_amount"); got != subtotal-want {
		t.Fatalf("payable_amount = %d, want %d", got, subtotal-want)
	}

	// The money that reaches the provider is the discounted one.
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/pay", customer,
		map[string]any{"provider": "mock"})
	if status != http.StatusOK && status != http.StatusCreated {
		t.Fatalf("pay: HTTP %d %v", status, body["message"])
	}
	var charged int64
	h.must(h.db.Raw(`SELECT amount FROM payments WHERE booking_id = ? ORDER BY created_at DESC LIMIT 1`,
		bookingID).Scan(&charged).Error)
	if charged != subtotal-want {
		t.Fatalf("payments.amount = %d, want the DISCOUNTED %d - the gateway must not charge full price",
			charged, subtotal-want)
	}
}

// Removing a code restores the full price and gives the redemption back, so a
// limited code is not burned by a customer who changed their mind.
func TestHTTP_DiscountRemoveRestoresPriceAndRedemption(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	id := createCode(t, h, admin, map[string]any{
		"code": "ONCE", "kind": "amount", "value": 10000, "max_uses": 1,
	})
	bookingID := holdSeats(t, h, customer, "A1")

	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "ONCE"})
	if status != http.StatusOK {
		t.Fatalf("apply: HTTP %d %v", status, body["message"])
	}
	subtotal := num(t, dataMap(body), "subtotal")

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/discounts/"+id, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("read code: HTTP %d %v", status, body["message"])
	}
	if got := num(t, dataMap(body), "used_count"); got != 1 {
		t.Fatalf("used_count = %d after applying, want 1", got)
	}

	status, _, body = h.call(http.MethodDelete, "/api/v1/orders/"+bookingID+"/discount", customer, nil)
	if status != http.StatusOK {
		t.Fatalf("remove: HTTP %d %v", status, body["message"])
	}
	cleared := dataMap(body)
	if got := num(t, cleared, "discount"); got != 0 {
		t.Fatalf("discount = %d after removal, want 0", got)
	}
	if got := num(t, cleared, "payable"); got != subtotal {
		t.Fatalf("payable = %d after removal, want the full %d", got, subtotal)
	}

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/discounts/"+id, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("re-read code: HTTP %d %v", status, body["message"])
	}
	if got := num(t, dataMap(body), "used_count"); got != 0 {
		t.Fatalf("used_count = %d after removal, want the redemption handed back", got)
	}
}

// max_uses is enforced in the UPDATE's WHERE, not by a prior SELECT, so the
// last remaining use cannot be handed to two orders.
func TestHTTP_DiscountExhaustsAtMaxUses(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	first, _ := h.login(models.RoleCustomer)
	second, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{
		"code": "ONLYONE", "kind": "amount", "value": 5000, "max_uses": 1,
	})

	firstBooking := holdSeats(t, h, first, "A1")
	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+firstBooking+"/discount", first,
		map[string]any{"code": "ONLYONE"})
	if status != http.StatusOK {
		t.Fatalf("first apply: HTTP %d %v", status, body["message"])
	}

	secondBooking := holdSeats(t, h, second, "A2")
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+secondBooking+"/discount", second,
		map[string]any{"code": "ONLYONE"})
	if status != http.StatusBadRequest {
		t.Fatalf("second apply: HTTP %d %v, want 400 (fully redeemed)", status, body["message"])
	}
}

// Every refusal is its own sentence because they all share code 40001 and the
// customer can only be told apart by the message. The one deliberate collision:
// an UNKNOWN code answers the same as a DISABLED one, so the endpoint cannot be
// used to find out which codes exist.
func TestHTTP_DiscountRefusals(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)
	other, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{
		"code": "DISABLED", "kind": "percent", "value": 10, "active": false,
	})
	createCode(t, h, admin, map[string]any{
		"code": "BIGSPEND", "kind": "percent", "value": 10, "min_order": 99_000_000,
	})
	createCode(t, h, admin, map[string]any{"code": "GOOD", "kind": "percent", "value": 10})

	bookingID := holdSeats(t, h, customer, "A1")

	_, _, unknown := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "NO-SUCH-CODE"})
	_, _, disabled := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "DISABLED"})
	if unknown["message"] != disabled["message"] {
		t.Fatalf("unknown (%v) and disabled (%v) must answer the SAME message, or the endpoint enumerates codes",
			unknown["message"], disabled["message"])
	}

	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "BIGSPEND"})
	if status != http.StatusBadRequest {
		t.Fatalf("below minimum: HTTP %d %v, want 400", status, body["message"])
	}
	if body["message"] == unknown["message"] {
		t.Fatalf("a below-minimum code must say so, not reuse the generic message")
	}

	// Another customer's order is 403, matching POST /orders/:id/refresh.
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", other,
		map[string]any{"code": "GOOD"})
	if status != http.StatusForbidden {
		t.Fatalf("someone else's order: HTTP %d %v, want 403", status, body["message"])
	}

	// One code at a time: a second apply is refused rather than silently swapping.
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "GOOD"})
	if status != http.StatusOK {
		t.Fatalf("first apply: HTTP %d %v", status, body["message"])
	}
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "GOOD"})
	if status != http.StatusConflict {
		t.Fatalf("second apply: HTTP %d %v, want 409", status, body["message"])
	}

	// Removing when there is nothing to remove is a 400, not a silent success.
	status, _, _ = h.call(http.MethodDelete, "/api/v1/orders/"+bookingID+"/discount", customer, nil)
	if status != http.StatusOK {
		t.Fatalf("remove the applied code: HTTP %d", status)
	}
	status, _, body = h.call(http.MethodDelete, "/api/v1/orders/"+bookingID+"/discount", customer, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("remove again: HTTP %d %v, want 400", status, body["message"])
	}
}

// A percentage code's cap is the difference between a 10% discount and a
// runaway one on an expensive order.
func TestHTTP_DiscountPercentRespectsItsCap(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{
		"code": "CAPPED", "kind": "percent", "value": 50, "max_discount": 1000,
	})
	bookingID := holdSeats(t, h, customer, "A1", "A2")

	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "CAPPED"})
	if status != http.StatusOK {
		t.Fatalf("apply: HTTP %d %v", status, body["message"])
	}
	if got := num(t, dataMap(body), "discount"); got != 1000 {
		t.Fatalf("discount = %d, want it capped at 1000 rather than 50%% of the order", got)
	}
}

// The catalogue is ADMIN-ONLY, unlike the concession catalogue next door: a code
// moves revenue, so it is a pricing decision rather than counter work.
func TestHTTP_DiscountCatalogueIsAdminOnly(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	id := createCode(t, h, admin, map[string]any{"code": "ADMINONLY", "kind": "percent", "value": 5})

	cases := []struct {
		name, method, path, token string
		body                      any
		want                      int
	}{
		{"staff cannot list", http.MethodGet, "/api/v1/admin/discounts", staff, nil, http.StatusForbidden},
		{"staff cannot create", http.MethodPost, "/api/v1/admin/discounts", staff,
			map[string]any{"code": "NOPE", "kind": "percent", "value": 5}, http.StatusForbidden},
		{"customer cannot list", http.MethodGet, "/api/v1/admin/discounts", customer, nil, http.StatusForbidden},
		{"anonymous cannot list", http.MethodGet, "/api/v1/admin/discounts", "", nil, http.StatusUnauthorized},
		{"admin lists", http.MethodGet, "/api/v1/admin/discounts", admin, nil, http.StatusOK},
		{"admin patches", http.MethodPatch, "/api/v1/admin/discounts/" + id, admin,
			map[string]any{"active": false}, http.StatusOK},
		{"empty patch is refused", http.MethodPatch, "/api/v1/admin/discounts/" + id, admin,
			map[string]any{}, http.StatusBadRequest},
		{"duplicate code is refused", http.MethodPost, "/api/v1/admin/discounts", admin,
			map[string]any{"code": "adminonly", "kind": "percent", "value": 5}, http.StatusConflict},
		{"a percentage over 100 is refused", http.MethodPost, "/api/v1/admin/discounts", admin,
			map[string]any{"code": "TOOMUCH", "kind": "percent", "value": 150}, http.StatusBadRequest},
		{"a cap on a flat code is refused", http.MethodPost, "/api/v1/admin/discounts", admin,
			map[string]any{"code": "FLATCAP", "kind": "amount", "value": 5000, "max_discount": 100}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, body := h.call(tc.method, tc.path, tc.token, tc.body)
			if status != tc.want {
				t.Fatalf("HTTP %d, want %d (%v)", status, tc.want, body["message"])
			}
		})
	}
}
