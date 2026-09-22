package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// createCombo adds one catalogue product through the operator API and returns its id.
func createCombo(t *testing.T, h *httpEnv, token string, payload map[string]any) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/concessions", token, payload)
	if status != http.StatusCreated {
		t.Fatalf("create concession: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["id"].(string)
	if id == "" {
		t.Fatalf("create concession returned no id: %v", body)
	}
	return id
}

// comboNames pulls the `name` of every row out of a paged operator listing.
func comboNames(t *testing.T, body map[string]any) []string {
	t.Helper()
	items, _ := dataMap(body)["items"].([]any)
	out := make([]string, 0, len(items))
	for _, raw := range items {
		row, _ := raw.(map[string]any)
		name, _ := row["name"].(string)
		out = append(out, name)
	}
	return out
}

// The public catalogue and the operator catalogue are deliberately different
// lists: GET /combos is what a customer may buy, so it must never leak a product
// taken off sale, while the operator list must show it — otherwise there is no
// way to put it back.
func TestHTTP_ConcessionInactiveHiddenFromCustomersButVisibleToOperator(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	onSale := createCombo(t, h, admin, map[string]any{"name": "Bap rang bo", "price": 59000})
	offSale := createCombo(t, h, admin, map[string]any{"name": "Combo theo mua", "price": 149000, "active": false})

	status, _, body := h.call(http.MethodGet, "/api/v1/combos", "", nil)
	if status != http.StatusOK {
		t.Fatalf("public list: HTTP %d %v", status, body["message"])
	}
	// The public endpoint answers a BARE ARRAY, not the paged {items, meta} shape.
	publicRows, _ := body["data"].([]any)
	for _, raw := range publicRows {
		row, _ := raw.(map[string]any)
		if row["id"] == offSale {
			t.Fatalf("GET /combos leaked an inactive product: %v", row["name"])
		}
	}
	if len(publicRows) != 1 {
		t.Fatalf("public list = %d rows, want 1 (only the on-sale product)", len(publicRows))
	}

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/concessions", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("operator list: HTTP %d %v", status, body["message"])
	}
	if names := comboNames(t, body); len(names) != 2 {
		t.Fatalf("operator list = %v, want both products including the inactive one", names)
	}

	// active=false narrows to exactly the retired product; the filter is a Go
	// pointer, so "absent" and "false" must not collapse into the same query.
	status, _, body = h.call(http.MethodGet, "/api/v1/admin/concessions?active=false", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("filtered list: HTTP %d %v", status, body["message"])
	}
	names := comboNames(t, body)
	if len(names) != 1 || names[0] != "Combo theo mua" {
		t.Fatalf("active=false gave %v, want only the retired product", names)
	}
	_ = onSale
}

// PATCH is a partial update built from a map, not a struct. That is the whole
// point: `active: false` and `price: 0` are Go zero values, and a struct-based
// GORM update would silently skip them, leaving a product on sale that the
// operator just took down.
func TestHTTP_ConcessionPatchWritesZeroValuesAndKeepsTheRest(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	id := createCombo(t, h, admin, map[string]any{
		"name": "Snack", "description": "mo ta goc", "price": 45000,
	})

	// Price only: description and active must survive untouched.
	status, _, body := h.call(http.MethodPatch, "/api/v1/admin/concessions/"+id, admin,
		map[string]any{"price": 50000})
	if status != http.StatusOK {
		t.Fatalf("patch price: HTTP %d %v", status, body["message"])
	}
	row := dataMap(body)
	if row["description"] != "mo ta goc" {
		t.Fatalf("description = %v, want it untouched by a price-only patch", row["description"])
	}
	if row["active"] != true {
		t.Fatalf("active = %v, want it untouched by a price-only patch", row["active"])
	}

	// active:false — the zero value a struct update would drop.
	status, _, body = h.call(http.MethodPatch, "/api/v1/admin/concessions/"+id, admin,
		map[string]any{"active": false})
	if status != http.StatusOK {
		t.Fatalf("patch active: HTTP %d %v", status, body["message"])
	}
	if dataMap(body)["active"] != false {
		t.Fatalf("active = %v, want false to be written", dataMap(body)["active"])
	}

	// price:0 — the other zero value. Re-read rather than trusting the echo:
	// the write and the read disagreeing is exactly how the hall-layout bug hid.
	status, _, body = h.call(http.MethodPatch, "/api/v1/admin/concessions/"+id, admin,
		map[string]any{"price": 0})
	if status != http.StatusOK {
		t.Fatalf("patch price 0: HTTP %d %v", status, body["message"])
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/admin/concessions/"+id, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("re-read: HTTP %d %v", status, body["message"])
	}
	row = dataMap(body)
	if row["price"] != float64(0) {
		t.Fatalf("re-read price = %v, want 0 to have been persisted", row["price"])
	}
	if row["active"] != false {
		t.Fatalf("re-read active = %v, want false to have been persisted", row["active"])
	}

	// A body that names no field changes nothing, so it is rejected rather than
	// issuing a no-op UPDATE — same contract as PATCH /admin/users/:id.
	status, _, body = h.call(http.MethodPatch, "/api/v1/admin/concessions/"+id, admin, map[string]any{})
	if status != http.StatusBadRequest {
		t.Fatalf("empty patch: HTTP %d %v, want 400", status, body["message"])
	}
}

// Retiring a product must not destroy the receipts that reference it: the FK on
// combo_order_items points at this row, and the delete is soft for that reason.
func TestHTTP_ConcessionDeleteIsSoftAndLeavesTheCatalogue(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	id := createCombo(t, h, admin, map[string]any{"name": "Combo sap go", "price": 99000})

	status, _, _ := h.call(http.MethodDelete, "/api/v1/admin/concessions/"+id, admin, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete: HTTP %d, want 204", status)
	}

	// Gone from both lists and from the detail read...
	status, _, body := h.call(http.MethodGet, "/api/v1/admin/concessions/"+id, admin, nil)
	if status != http.StatusNotFound {
		t.Fatalf("read after delete: HTTP %d %v, want 404", status, body["message"])
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/admin/concessions", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("list after delete: HTTP %d %v", status, body["message"])
	}
	if names := comboNames(t, body); len(names) != 0 {
		t.Fatalf("list after delete = %v, want empty", names)
	}

	// ...but still present in the table, which is what keeps old orders readable.
	if n := h.count("SELECT count(*) FROM concession_items WHERE id = ?", id); n != 1 {
		t.Fatalf("rows for the deleted product = %d, want 1 (soft delete, not a purge)", n)
	}
}

// The catalogue is OPERATOR scope (admin AND staff), matching halls and
// showtimes rather than the admin-only /admin/users and /admin/reports: putting
// popcorn back on sale is counter work. Customers must never reach it at all.
func TestHTTP_ConcessionCatalogueIsOperatorScope(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	id := createCombo(t, h, admin, map[string]any{"name": "Bap rang bo", "price": 59000})

	cases := []struct {
		name, method, path, token string
		body                      any
		want                      int
	}{
		{"staff lists", http.MethodGet, "/api/v1/admin/concessions", staff, nil, http.StatusOK},
		{"staff creates", http.MethodPost, "/api/v1/admin/concessions", staff,
			map[string]any{"name": "Nuoc suoi", "price": 20000}, http.StatusCreated},
		{"staff patches", http.MethodPatch, "/api/v1/admin/concessions/" + id, staff,
			map[string]any{"price": 60000}, http.StatusOK},
		{"customer is refused", http.MethodGet, "/api/v1/admin/concessions", customer, nil, http.StatusForbidden},
		{"customer cannot create", http.MethodPost, "/api/v1/admin/concessions", customer,
			map[string]any{"name": "Free", "price": 0}, http.StatusForbidden},
		{"anonymous is refused", http.MethodGet, "/api/v1/admin/concessions", "", nil, http.StatusUnauthorized},
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
