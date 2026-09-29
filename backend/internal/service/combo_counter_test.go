package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

func counterSell(t *testing.T, h *httpEnv, token string, payload map[string]any) (int, map[string]any) {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/staff/combo-orders", token, payload)
	return status, body
}

// Walk-in sale lands collected, never in the pickup queue.
func TestHTTP_CounterComboSell(t *testing.T) {
	h := newHTTPEnv(t)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	popcorn := createCombo(t, h, staff, map[string]any{"name": "Bap rang bo", "price": 59000})
	soda := createCombo(t, h, staff, map[string]any{"name": "Nuoc ngot", "price": 30000})

	status, body := counterSell(t, h, staff, map[string]any{
		"items":           []any{map[string]any{"combo_id": popcorn, "quantity": 2}, map[string]any{"combo_id": soda, "quantity": 1}},
		"pay_method":      "cash",
		"customer_name":   "Khach vang lai",
	})
	if status != http.StatusCreated {
		t.Fatalf("counter sell: HTTP %d %v", status, body["message"])
	}
	got := dataMap(body)
	if got["status"] != models.ComboOrderCollected {
		t.Fatalf("status = %v, want %v", got["status"], models.ComboOrderCollected)
	}
	if got["sold_channel"] != models.SoldChannelCounter || got["pay_method"] != models.PayMethodCash {
		t.Fatalf("channel/payment = %v/%v", got["sold_channel"], got["pay_method"])
	}
	if total, _ := got["total"].(float64); total != 2*59000+30000 {
		t.Fatalf("total = %v, want %d", got["total"], 2*59000+30000)
	}
	if _, hasUser := got["user_id"]; hasUser {
		t.Fatalf("walk-in order carries a user_id: %v", got["user_id"])
	}

	// A customer calling the staff endpoint gets 403, not a silent no-op.
	status, _ = counterSell(t, h, customer, map[string]any{
		"items":      []any{map[string]any{"combo_id": popcorn, "quantity": 1}},
		"pay_method": "pos",
	})
	if status != http.StatusForbidden {
		t.Fatalf("customer counter sell: HTTP %d, want 403", status)
	}

	// Refusals: unknown/inactive product, empty lines, unknown takings method.
	ghost := "00000000-0000-0000-0000-000000000000"
	for name, payload := range map[string]map[string]any{
		"unknown product": {"items": []any{map[string]any{"combo_id": ghost, "quantity": 1}}, "pay_method": "cash"},
		"empty items":     {"items": []any{}, "pay_method": "cash"},
		"bad pay method":  {"items": []any{map[string]any{"combo_id": popcorn, "quantity": 1}}, "pay_method": "transfer"},
	} {
		if status, _ := counterSell(t, h, staff, payload); status < 400 {
			t.Fatalf("%s: HTTP %d, want 4xx", name, status)
		}
	}

	offSale := createCombo(t, h, staff, map[string]any{"name": "Combo nguoi mua", "price": 99000, "active": false})
	if status, _ := counterSell(t, h, staff, map[string]any{
		"items": []any{map[string]any{"combo_id": offSale, "quantity": 1}}, "pay_method": "cash",
	}); status < 400 {
		t.Fatalf("off-sale product: HTTP %d, want 4xx", status)
	}
}
