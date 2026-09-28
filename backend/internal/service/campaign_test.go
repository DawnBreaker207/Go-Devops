package service_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// createArticle adds one article through the operator API and returns its id.
func createArticle(t *testing.T, h *httpEnv, token string, payload map[string]any) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/articles", token, payload)
	if status != http.StatusCreated {
		t.Fatalf("create article: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["id"].(string)
	if id == "" {
		t.Fatalf("create article returned no id: %v", body)
	}
	return id
}

// createCampaign creates a campaign through the admin API and returns its id.
func createCampaign(t *testing.T, h *httpEnv, admin string, payload map[string]any) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/campaigns", admin, payload)
	if status != http.StatusCreated {
		t.Fatalf("create campaign: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["id"].(string)
	if id == "" {
		t.Fatalf("create campaign returned no id: %v", body)
	}
	return id
}

// The full section-9 happy path: draft -> attach code+combo+article -> active
// in window -> visible via GET /campaigns -> applying its code deducts
// Payable() correctly.
func TestHTTP_CampaignFullLifecycle(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	now := time.Now().UTC()
	campaignID := createCampaign(t, h, admin, map[string]any{
		"name":      "Tet 2027",
		"starts_at": now.Add(-time.Hour).Format(time.RFC3339),
		"ends_at":   now.Add(24 * time.Hour).Format(time.RFC3339),
		"active":    true,
	})

	codeID := createCode(t, h, admin, map[string]any{
		"code": "TET2027", "kind": "percent", "value": 10,
	})
	comboID := createCombo(t, h, admin, map[string]any{"name": "Combo Tet", "price": 100000})
	articleID := createArticle(t, h, admin, map[string]any{
		"title": "Uu dai Tet 2027", "content": "noi dung", "type": "promotion", "status": "published",
	})

	if status, _, body := h.call(http.MethodPost,
		fmt.Sprintf("/api/v1/admin/campaigns/%s/discount-codes/%s", campaignID, codeID), admin, nil); status != http.StatusOK {
		t.Fatalf("attach code: HTTP %d %v", status, body["message"])
	}
	if status, _, body := h.call(http.MethodPost,
		fmt.Sprintf("/api/v1/admin/campaigns/%s/combos/%s", campaignID, comboID), admin,
		map[string]any{"promo_price": 79000}); status != http.StatusOK {
		t.Fatalf("attach combo: HTTP %d %v", status, body["message"])
	}
	if status, _, body := h.call(http.MethodPost,
		fmt.Sprintf("/api/v1/admin/campaigns/%s/articles/%s", campaignID, articleID), admin, nil); status != http.StatusOK {
		t.Fatalf("attach article: HTTP %d %v", status, body["message"])
	}

	// Visible publicly, with the three links folded in.
	status, _, body := h.call(http.MethodGet, "/api/v1/campaigns/"+campaignID, "", nil)
	if status != http.StatusOK {
		t.Fatalf("public get: HTTP %d %v", status, body["message"])
	}
	data := dataMap(body)
	codes, _ := data["discount_codes"].([]any)
	if len(codes) != 1 || codes[0].(map[string]any)["code"] != "TET2027" {
		t.Fatalf("public campaign discount_codes = %v", codes)
	}
	combos, _ := data["combos"].([]any)
	if len(combos) != 1 || combos[0].(map[string]any)["combo_id"] != comboID {
		t.Fatalf("public campaign combos = %v", combos)
	}
	articles, _ := data["articles"].([]any)
	if len(articles) != 1 || articles[0].(map[string]any)["id"] != articleID {
		t.Fatalf("public campaign articles = %v", articles)
	}

	// Shows up in the public list too.
	status, _, body = h.call(http.MethodGet, "/api/v1/campaigns", "", nil)
	if status != http.StatusOK {
		t.Fatalf("public list: HTTP %d %v", status, body["message"])
	}
	items, _ := dataMap(body)["items"].([]any)
	found := false
	for _, it := range items {
		if it.(map[string]any)["id"] == campaignID {
			found = true
		}
	}
	if !found {
		t.Fatalf("campaign %s missing from public list: %v", campaignID, items)
	}

	// Applying the campaign's code still deducts Payable() correctly.
	bookingID := holdSeats(t, h, customer, "A1", "A2")
	status, _, body = h.call(http.MethodGet, "/api/v1/orders/"+bookingID, customer, nil)
	if status != http.StatusOK {
		t.Fatalf("read order: HTTP %d %v", status, body["message"])
	}
	subtotal := num(t, dataMap(body), "total_amount")

	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
		map[string]any{"code": "TET2027"})
	if status != http.StatusOK {
		t.Fatalf("apply campaign code: HTTP %d %v", status, body["message"])
	}
	applied := dataMap(body)
	want := subtotal / 10
	if got := num(t, applied, "discount"); got != want {
		t.Fatalf("discount = %d, want %d", got, want)
	}
	if got := num(t, applied, "payable"); got != subtotal-want {
		t.Fatalf("payable = %d, want %d", got, subtotal-want)
	}
}

// The single most important acceptance criterion: the SAME user applying the
// SAME code a second time (on a second order) is rejected with the new reason
// and an English message, whether or not the code belongs to a campaign.
func TestHTTP_CampaignDiscountRejectsSecondRedemptionBySameUser(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{"code": "ONEEACH", "kind": "amount", "value": 5000})

	firstBooking := holdSeats(t, h, customer, "A1")
	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+firstBooking+"/discount", customer,
		map[string]any{"code": "ONEEACH"})
	if status != http.StatusOK {
		t.Fatalf("first apply: HTTP %d %v", status, body["message"])
	}

	secondBooking := holdSeats(t, h, customer, "A2")
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+secondBooking+"/discount", customer,
		map[string]any{"code": "ONEEACH"})
	if status != http.StatusConflict {
		t.Fatalf("second apply on a DIFFERENT order: HTTP %d %v, want 409", status, body["message"])
	}
	if body["reason"] != "discount_already_redeemed" {
		t.Fatalf("reason = %v, want discount_already_redeemed", body["reason"])
	}
	if body["message"] == "" || body["message"] == nil {
		t.Fatalf("message missing: %v", body)
	}
}

// A user MAY apply two DIFFERENT codes from the same campaign - the guard is
// keyed per (user, code), not per (user, campaign).
func TestHTTP_CampaignAllowsDifferentCodesFromSameCampaignForSameUser(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	now := time.Now().UTC()
	campaignID := createCampaign(t, h, admin, map[string]any{
		"name": "Multi-code campaign", "starts_at": now.Add(-time.Hour).Format(time.RFC3339),
		"ends_at": now.Add(24 * time.Hour).Format(time.RFC3339), "active": true,
	})
	codeA := createCode(t, h, admin, map[string]any{"code": "CAMPA", "kind": "amount", "value": 1000})
	codeB := createCode(t, h, admin, map[string]any{"code": "CAMPB", "kind": "amount", "value": 1000})
	for _, id := range []string{codeA, codeB} {
		if status, _, body := h.call(http.MethodPost,
			fmt.Sprintf("/api/v1/admin/campaigns/%s/discount-codes/%s", campaignID, id), admin, nil); status != http.StatusOK {
			t.Fatalf("attach %s: HTTP %d %v", id, status, body["message"])
		}
	}

	b1 := holdSeats(t, h, customer, "A1")
	if status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+b1+"/discount", customer,
		map[string]any{"code": "CAMPA"}); status != http.StatusOK {
		t.Fatalf("apply CAMPA: HTTP %d %v", status, body["message"])
	}
	b2 := holdSeats(t, h, customer, "A2")
	if status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+b2+"/discount", customer,
		map[string]any{"code": "CAMPB"}); status != http.StatusOK {
		t.Fatalf("apply CAMPB (a DIFFERENT code from the same campaign): HTTP %d %v", status, body["message"])
	}
}

// Outside the campaign's window, or with the campaign switched off, applying
// its code is refused even though the code itself is active with no window
// of its own.
func TestHTTP_CampaignWindowGatesItsCode(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	now := time.Now().UTC()

	notStartedYet := createCampaign(t, h, admin, map[string]any{
		"name": "Future campaign", "starts_at": now.Add(24 * time.Hour).Format(time.RFC3339),
		"ends_at": now.Add(48 * time.Hour).Format(time.RFC3339), "active": true,
	})
	futureCode := createCode(t, h, admin, map[string]any{"code": "FUTURE", "kind": "amount", "value": 1000})
	h.call(http.MethodPost, fmt.Sprintf("/api/v1/admin/campaigns/%s/discount-codes/%s", notStartedYet, futureCode), admin, nil)

	inactive := createCampaign(t, h, admin, map[string]any{
		"name": "Switched off", "starts_at": now.Add(-time.Hour).Format(time.RFC3339),
		"ends_at": now.Add(time.Hour).Format(time.RFC3339), "active": false,
	})
	offCode := createCode(t, h, admin, map[string]any{"code": "SWITCHEDOFF", "kind": "amount", "value": 1000})
	h.call(http.MethodPost, fmt.Sprintf("/api/v1/admin/campaigns/%s/discount-codes/%s", inactive, offCode), admin, nil)

	for _, code := range []string{"FUTURE", "SWITCHEDOFF"} {
		bookingID := holdSeats(t, h, customer, "A1")
		status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/discount", customer,
			map[string]any{"code": code})
		if status != http.StatusBadRequest {
			t.Fatalf("apply %s: HTTP %d %v, want 400 (campaign_inactive)", code, status, body["message"])
		}
		if body["reason"] != "campaign_inactive" {
			t.Fatalf("apply %s: reason = %v, want campaign_inactive", code, body["reason"])
		}
		// Cancel the order so its seat is free for the next case.
		h.call(http.MethodPost, "/api/v1/orders/"+bookingID+"/cancel", customer, nil)
	}
}

// Remove hands the redemption back, so the SAME user can apply the SAME code
// again on a new order.
func TestHTTP_CampaignRemoveRestoresRedemption(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{"code": "GIVEBACK", "kind": "amount", "value": 1000})

	first := holdSeats(t, h, customer, "A1")
	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+first+"/discount", customer,
		map[string]any{"code": "GIVEBACK"})
	if status != http.StatusOK {
		t.Fatalf("apply: HTTP %d %v", status, body["message"])
	}
	status, _, body = h.call(http.MethodDelete, "/api/v1/orders/"+first+"/discount", customer, nil)
	if status != http.StatusOK {
		t.Fatalf("remove: HTTP %d %v", status, body["message"])
	}

	second := holdSeats(t, h, customer, "A2")
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+second+"/discount", customer,
		map[string]any{"code": "GIVEBACK"})
	if status != http.StatusOK {
		t.Fatalf("re-apply after remove: HTTP %d %v, want 200 (the redemption must have been given back)", status, body["message"])
	}
}

// Concurrent double-apply by the SAME user: only one of two simultaneous
// requests may win, enforced by discount_redemptions' UNIQUE(user_id,
// discount_code_id) - no prior SELECT, same idiom as ClaimUse.
func TestHTTP_CampaignConcurrentSameUserOnlyOneWins(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{"code": "RACEME", "kind": "amount", "value": 1000, "max_uses": 10})

	first := holdSeats(t, h, customer, "A1")
	second := holdSeats(t, h, customer, "A2")

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	bookingIDs := []string{first, second}
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer wg.Done()
			status, _, _ := h.call(http.MethodPost, "/api/v1/orders/"+bookingIDs[i]+"/discount", customer,
				map[string]any{"code": "RACEME"})
			statuses[i] = status
		}(i)
	}
	wg.Wait()

	wins, losses := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusOK:
			wins++
		case http.StatusConflict:
			losses++
		default:
			t.Fatalf("unexpected status %d in %v", status, statuses)
		}
	}
	if wins != 1 || losses != 1 {
		t.Fatalf("statuses = %v, want exactly one 200 and one 409", statuses)
	}
}

// The global max_uses counter still works alongside the new per-user guard:
// two DIFFERENT users racing for the LAST global use still only lets one through.
func TestHTTP_CampaignGlobalMaxUsesStillWorksAlongsideRedemption(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	first, _ := h.login(models.RoleCustomer)
	second, _ := h.login(models.RoleCustomer)

	createCode(t, h, admin, map[string]any{"code": "LASTONE", "kind": "amount", "value": 1000, "max_uses": 1})

	firstBooking := holdSeats(t, h, first, "A1")
	status, _, body := h.call(http.MethodPost, "/api/v1/orders/"+firstBooking+"/discount", first,
		map[string]any{"code": "LASTONE"})
	if status != http.StatusOK {
		t.Fatalf("first user: HTTP %d %v", status, body["message"])
	}

	secondBooking := holdSeats(t, h, second, "A2")
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/"+secondBooking+"/discount", second,
		map[string]any{"code": "LASTONE"})
	if status != http.StatusBadRequest {
		t.Fatalf("second user: HTTP %d %v, want 400 (discount_exhausted, a DIFFERENT user hitting the GLOBAL limit)", status, body["message"])
	}
	if body["reason"] != "discount_exhausted" {
		t.Fatalf("reason = %v, want discount_exhausted", body["reason"])
	}
}

// Admin CRUD + attach/detach basics, and the group is ADMIN-ONLY like discounts.
func TestHTTP_CampaignAdminCRUDAndScope(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	now := time.Now().UTC()
	id := createCampaign(t, h, admin, map[string]any{
		"name": "CRUD campaign", "starts_at": now.Format(time.RFC3339), "ends_at": now.Add(time.Hour).Format(time.RFC3339),
	})

	if status, _, body := h.call(http.MethodGet, "/api/v1/admin/campaigns", staff, nil); status != http.StatusForbidden {
		t.Fatalf("staff list: HTTP %d %v, want 403", status, body["message"])
	}
	if status, _, body := h.call(http.MethodGet, "/api/v1/admin/campaigns", customer, nil); status != http.StatusForbidden {
		t.Fatalf("customer list: HTTP %d %v, want 403", status, body["message"])
	}

	status, _, body := h.call(http.MethodPatch, "/api/v1/admin/campaigns/"+id, admin, map[string]any{"active": true})
	if status != http.StatusOK || dataMap(body)["active"] != true {
		t.Fatalf("patch active: HTTP %d %v", status, body)
	}

	status, _, body = h.call(http.MethodPatch, "/api/v1/admin/campaigns/"+id, admin, map[string]any{})
	if status != http.StatusBadRequest {
		t.Fatalf("empty patch: HTTP %d %v, want 400", status, body["message"])
	}

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/campaigns/"+id, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("admin get: HTTP %d %v", status, body["message"])
	}

	status, _, _ = h.call(http.MethodDelete, "/api/v1/admin/campaigns/"+id, admin, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete: HTTP %d", status)
	}
	status, _, _ = h.call(http.MethodGet, "/api/v1/admin/campaigns/"+id, admin, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get after delete: HTTP %d, want 404", status)
	}
}

// Deleting a campaign sets its discount code's campaign_id back to NULL
// rather than orphaning or deleting the code - it becomes a standalone code.
func TestHTTP_CampaignDeleteDetachesItsDiscountCode(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	now := time.Now().UTC()
	campaignID := createCampaign(t, h, admin, map[string]any{
		"name": "Doomed campaign", "starts_at": now.Format(time.RFC3339), "ends_at": now.Add(time.Hour).Format(time.RFC3339),
	})
	codeID := createCode(t, h, admin, map[string]any{"code": "SURVIVOR", "kind": "amount", "value": 1000})
	h.call(http.MethodPost, fmt.Sprintf("/api/v1/admin/campaigns/%s/discount-codes/%s", campaignID, codeID), admin, nil)

	status, _, _ := h.call(http.MethodDelete, "/api/v1/admin/campaigns/"+campaignID, admin, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete campaign: HTTP %d", status)
	}

	status, _, body := h.call(http.MethodGet, "/api/v1/admin/discounts/"+codeID, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("read code after campaign delete: HTTP %d %v", status, body["message"])
	}
	if dataMap(body)["campaign_id"] != nil {
		t.Fatalf("campaign_id = %v after campaign delete, want nil (standalone again)", dataMap(body)["campaign_id"])
	}
}
