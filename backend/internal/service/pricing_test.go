package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// setShowStart pins h.showID's start/end to an ABSOLUTE UTC instant so a Quote
// test can target an exact weekday, time-of-day or calendar date instead of
// fighting the wall clock. buildEngine wires PricingService with time.UTC, so
// the instant passed here is exactly what Quote's day/time matching sees.
func setShowStart(t *testing.T, h *httpEnv, at time.Time) {
	t.Helper()
	h.must(h.db.Exec(`UPDATE showtimes SET start_at = ?, end_at = ? WHERE id = ?`,
		at, at.Add(2*time.Hour), h.showID).Error)
}

func setBasePrice(t *testing.T, h *httpEnv, admin, seatType string, price int64) {
	t.Helper()
	status, _, body := h.call(http.MethodPut, "/api/v1/admin/pricing/base", admin,
		map[string]any{"prices": map[string]any{seatType: price}})
	if status != http.StatusOK {
		t.Fatalf("set base price: HTTP %d %v", status, body["message"])
	}
}

func createRule(t *testing.T, h *httpEnv, admin string, payload map[string]any) string {
	t.Helper()
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/pricing/rules", admin, payload)
	if status != http.StatusCreated {
		t.Fatalf("create rule: HTTP %d %v", status, body["message"])
	}
	id, _ := dataMap(body)["id"].(string)
	if id == "" {
		t.Fatalf("create rule returned no id: %v", body)
	}
	return id
}

func quote(t *testing.T, h *httpEnv, showID, seatType string) map[string]any {
	t.Helper()
	status, _, body := h.call(http.MethodGet,
		"/api/v1/pricing/quote?showtime_id="+showID+"&seat_type="+seatType, "", nil)
	if status != http.StatusOK {
		t.Fatalf("quote: HTTP %d %v", status, body["message"])
	}
	return dataMap(body)
}

/* -------------------------------------------------------------------------- */
/* Base prices                                                                 */
/* -------------------------------------------------------------------------- */

func TestHTTP_PricingBasePriceCRUD(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)

	// Migration 000013 always leaves exactly 4 rows (one per seat type,
	// backfilled to 0 in a fresh test DB with no hall_prices yet).
	status, _, body := h.call(http.MethodGet, "/api/v1/admin/pricing/base", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("get base prices: HTTP %d %v", status, body["message"])
	}
	rows, _ := body["data"].([]any)
	if len(rows) != 4 {
		t.Fatalf("base prices = %d rows, want 4", len(rows))
	}

	setBasePrice(t, h, admin, "vip", 123000)

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/pricing/base", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("re-read: HTTP %d %v", status, body["message"])
	}
	rows, _ = body["data"].([]any)
	var vipPrice float64
	found := false
	for _, r := range rows {
		row, _ := r.(map[string]any)
		if row["seat_type"] == "vip" {
			vipPrice, found = row["price"].(float64), true
		}
	}
	if !found || int64(vipPrice) != 123000 {
		t.Fatalf("vip price = %v (found=%v), want 123000", vipPrice, found)
	}

	// Non-admin (customer) is refused.
	customer, _ := h.login(models.RoleCustomer)
	status, _, _ = h.call(http.MethodPut, "/api/v1/admin/pricing/base", customer,
		map[string]any{"prices": map[string]any{"vip": 1}})
	if status != http.StatusForbidden {
		t.Fatalf("customer set base price: HTTP %d, want 403", status)
	}

	// An invalid seat type is rejected rather than silently accepted.
	status, _, _ = h.call(http.MethodPut, "/api/v1/admin/pricing/base", admin,
		map[string]any{"prices": map[string]any{"gold": 1}})
	if status != http.StatusBadRequest {
		t.Fatalf("invalid seat type: HTTP %d, want 400", status)
	}
}

/* -------------------------------------------------------------------------- */
/* Rule CRUD, admin-only                                                       */
/* -------------------------------------------------------------------------- */

func TestHTTP_PricingRuleCRUD(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	// Non-admin can not create, list, get, update or delete a rule.
	status, _, _ := h.call(http.MethodPost, "/api/v1/admin/pricing/rules", customer,
		map[string]any{"name": "x", "adjust_kind": "fixed", "adjust_value": 1000})
	if status != http.StatusForbidden {
		t.Fatalf("customer create rule: HTTP %d, want 403", status)
	}
	status, _, _ = h.call(http.MethodGet, "/api/v1/admin/pricing/rules", customer, nil)
	if status != http.StatusForbidden {
		t.Fatalf("customer list rules: HTTP %d, want 403", status)
	}

	id := createRule(t, h, admin, map[string]any{
		"name": "Weekend surcharge", "adjust_kind": "percent", "adjust_value": 10, "priority": 5,
	})

	status, _, body := h.call(http.MethodGet, "/api/v1/admin/pricing/rules/"+id, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("get rule: HTTP %d %v", status, body["message"])
	}
	if got := dataMap(body)["name"]; got != "Weekend surcharge" {
		t.Fatalf("name = %v, want %q", got, "Weekend surcharge")
	}

	status, _, body = h.call(http.MethodPatch, "/api/v1/admin/pricing/rules/"+id, admin,
		map[string]any{"active": false, "priority": 9})
	if status != http.StatusOK {
		t.Fatalf("update rule: HTTP %d %v", status, body["message"])
	}
	if dataMap(body)["active"] != false {
		t.Fatalf("active = %v after update, want false", dataMap(body)["active"])
	}
	if got := num(t, dataMap(body), "priority"); got != 9 {
		t.Fatalf("priority = %d after update, want 9", got)
	}

	status, _, body = h.call(http.MethodGet, "/api/v1/admin/pricing/rules", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("list rules: HTTP %d %v", status, body["message"])
	}
	items, _ := dataMap(body)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("list = %d items, want 1", len(items))
	}

	status, _, body = h.call(http.MethodDelete, "/api/v1/admin/pricing/rules/"+id, admin, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete rule: HTTP %d %v", status, body["message"])
	}
	status, _, _ = h.call(http.MethodGet, "/api/v1/admin/pricing/rules/"+id, admin, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get deleted rule: HTTP %d, want 404", status)
	}
}

/* -------------------------------------------------------------------------- */
/* Quote matrix                                                                */
/* -------------------------------------------------------------------------- */

// nextWeekday returns the next instant at or after `from` that falls on `day`,
// at the given hour:minute UTC - deterministic regardless of what day the test
// suite happens to run on.
func nextWeekday(from time.Time, day time.Weekday, hour, minute int) time.Time {
	at := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, time.UTC)
	for at.Weekday() != day || !at.After(from) {
		at = at.AddDate(0, 0, 1)
	}
	return at
}

func TestHTTP_PricingQuote_DayOfWeekRule(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)

	sunday := nextWeekday(time.Now().UTC(), time.Sunday, 20, 0)
	saturday := sunday.AddDate(0, 0, -1)

	dow := 0 // Sunday, matching time.Weekday
	createRule(t, h, admin, map[string]any{
		"name": "Sunday surcharge", "day_of_week": dow,
		"adjust_kind": "fixed", "adjust_value": 20000,
	})

	setShowStart(t, h, sunday)
	got := quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 120000 {
		t.Fatalf("Sunday quote final = %d, want 120000", final)
	}
	applied, _ := got["applied"].([]any)
	if len(applied) != 1 {
		t.Fatalf("Sunday quote applied = %d rules, want 1: %v", len(applied), applied)
	}

	setShowStart(t, h, saturday)
	got = quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 100000 {
		t.Fatalf("Saturday quote final = %d, want the untouched base 100000", final)
	}
	applied, _ = got["applied"].([]any)
	if len(applied) != 0 {
		t.Fatalf("Saturday quote applied = %v, want none (rule is Sunday-only)", applied)
	}
}

func TestHTTP_PricingQuote_TimeWindowRule(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)

	createRule(t, h, admin, map[string]any{
		"name": "Evening surcharge", "start_time": "18:00", "end_time": "23:00",
		"adjust_kind": "fixed", "adjust_value": 15000,
	})

	base := time.Now().UTC().Truncate(24 * time.Hour)
	inWindow := base.Add(19 * time.Hour)
	outOfWindow := base.Add(10 * time.Hour)

	setShowStart(t, h, inWindow)
	got := quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 115000 {
		t.Fatalf("in-window quote final = %d, want 115000", final)
	}

	setShowStart(t, h, outOfWindow)
	got = quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 100000 {
		t.Fatalf("out-of-window quote final = %d, want the untouched base 100000", final)
	}
}

func TestHTTP_PricingQuote_SpecificDateRule(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)

	holiday := time.Now().UTC().AddDate(0, 0, 10).Truncate(24 * time.Hour).Add(20 * time.Hour)
	otherDay := holiday.AddDate(0, 0, 1)

	createRule(t, h, admin, map[string]any{
		"name": "New Year surcharge", "specific_date": holiday.Format("2006-01-02"),
		"adjust_kind": "fixed", "adjust_value": 50000,
	})

	setShowStart(t, h, holiday)
	got := quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 150000 {
		t.Fatalf("holiday quote final = %d, want 150000", final)
	}

	setShowStart(t, h, otherDay)
	got = quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 100000 {
		t.Fatalf("non-holiday quote final = %d, want the untouched base 100000", final)
	}
}

// Two overlapping rules apply in priority order (highest first); percent
// compounds against the RUNNING total, so a +10% rule applied AFTER a +20000
// fixed rule takes 10% of (base+20000), not of the base alone.
func TestHTTP_PricingQuote_PriorityOrderAndPercentVsFixed(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)

	createRule(t, h, admin, map[string]any{
		"name": "Fixed surcharge (lower priority)", "adjust_kind": "fixed", "adjust_value": 20000, "priority": 1,
	})
	createRule(t, h, admin, map[string]any{
		"name": "Percent surcharge (higher priority)", "adjust_kind": "percent", "adjust_value": 10, "priority": 5,
	})

	got := quote(t, h, h.showID, "standard")
	// running = 100000 -> +10% (priority 5 first) = 110000 -> +20000 (priority 1) = 130000
	if final := num(t, got, "final"); final != 130000 {
		t.Fatalf("final = %d, want 130000 (percent-then-fixed by priority)", final)
	}
	applied, _ := got["applied"].([]any)
	if len(applied) != 2 {
		t.Fatalf("applied = %d rules, want 2: %v", len(applied), applied)
	}
	first, _ := applied[0].(map[string]any)
	if first["adjust_kind"] != "percent" {
		t.Fatalf("applied[0] = %v, want the higher-priority percent rule first", first)
	}
}

func TestHTTP_PricingQuote_InactiveRuleIgnored(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)

	id := createRule(t, h, admin, map[string]any{
		"name": "Disabled rule", "adjust_kind": "fixed", "adjust_value": 99999,
	})
	status, _, body := h.call(http.MethodPatch, "/api/v1/admin/pricing/rules/"+id, admin,
		map[string]any{"active": false})
	if status != http.StatusOK {
		t.Fatalf("disable rule: HTTP %d %v", status, body["message"])
	}

	got := quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 100000 {
		t.Fatalf("final = %d, want the untouched base 100000 (rule is inactive)", final)
	}
	applied, _ := got["applied"].([]any)
	if len(applied) != 0 {
		t.Fatalf("applied = %v, want none", applied)
	}
}

func TestHTTP_PricingQuote_FloorsAtZero(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 50000)

	createRule(t, h, admin, map[string]any{
		"name": "Deep discount", "adjust_kind": "fixed", "adjust_value": -999999,
	})

	got := quote(t, h, h.showID, "standard")
	if final := num(t, got, "final"); final != 0 {
		t.Fatalf("final = %d, want floored at 0", final)
	}
	if base := num(t, got, "base"); base != 50000 {
		t.Fatalf("base = %d, want the untouched 50000 (only final is floored)", base)
	}
}

func TestHTTP_PricingQuote_UnknownShowtimeIs404(t *testing.T) {
	h := newHTTPEnv(t)
	status, _, body := h.call(http.MethodGet,
		"/api/v1/pricing/quote?showtime_id=00000000-0000-0000-0000-000000000000&seat_type=standard", "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("unknown showtime: HTTP %d %v, want 404", status, body["message"])
	}
}

/* -------------------------------------------------------------------------- */
/* Phase 2: hold, counter-sell and the seatmap all read the SAME engine       */
/* (PLAN_CAMPAIGN.md section 11.3) — a rule that fires for the showtime's     */
/* actual start time must produce the adjusted price everywhere: the seatmap  */
/* (before any seat is picked), the hold snapshot and the counter-sell        */
/* snapshot, all via PricingService.Quote/QuotePrices's own rule-matching     */
/* code, never a second copy of it.                                          */
/* -------------------------------------------------------------------------- */

func TestHTTP_PricingRule_AppliesAtHoldSeatmapAndCounterSell(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	setBasePrice(t, h, admin, "standard", 100000)
	setBasePrice(t, h, admin, "vip", 150000)

	sunday := nextWeekday(time.Now().UTC(), time.Sunday, 20, 0)
	dow := 0
	createRule(t, h, admin, map[string]any{
		"name": "Sunday surcharge", "day_of_week": dow,
		"adjust_kind": "fixed", "adjust_value": 20000,
	})
	setShowStart(t, h, sunday)

	// Seatmap: the adjusted price shows up BEFORE any seat is held.
	m, err := h.showtimes.SeatMap(h.ctx, h.showID)
	h.must(err)
	if m.Prices[models.SeatStandard] != 120000 || m.Prices[models.SeatVIP] != 170000 {
		t.Fatalf("seatmap prices = %+v, want standard=120000 vip=170000", m.Prices)
	}
	var seatA2Price int64
	for _, s := range m.Seats {
		if s.Label == "A2" {
			seatA2Price = s.Price
		}
	}
	if seatA2Price != 120000 {
		t.Fatalf("seatmap A2 (standard) price = %d, want 120000", seatA2Price)
	}

	// Hold: the same adjusted price is snapshotted into booking_seats.price -
	// finalizeTx/Payable/refund read only that snapshot from here on.
	held := h.mustHold(h.users[0], "A2")
	if held.Seats[0].Price != 120000 {
		t.Fatalf("hold response price = %d, want 120000", held.Seats[0].Price)
	}
	var snapshotPrice int64
	h.must(h.db.Raw(`SELECT price FROM booking_seats WHERE booking_id = ?`, held.BookingID).Scan(&snapshotPrice).Error)
	if snapshotPrice != 120000 {
		t.Fatalf("booking_seats snapshot = %d, want 120000", snapshotPrice)
	}

	// Counter-sell on the same (rule-matching) showtime: same engine, a
	// different seat type.
	res, err := h.svc.CounterSell(h.ctx, dto.CounterSellRequest{
		ShowID: h.showID, SeatIDs: h.ids("B2"),
		CustomerName: "Walk-in", CustomerPhone: "0900000000",
	})
	h.must(err)
	if res.TotalAmount != 170000 {
		t.Fatalf("counter sell total = %d, want 170000 (vip base 150000 + rule 20000)", res.TotalAmount)
	}
}
