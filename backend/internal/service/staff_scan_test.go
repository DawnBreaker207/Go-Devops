package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Scanner resolves an order from a ticket id or QR code.
func TestHTTP_OrderByTicketCode(t *testing.T) {
	h := newHTTPEnv(t)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	sold, err := h.svc.CounterSell(h.ctx, dto.CounterSellRequest{
		ShowID: h.showID, SeatIDs: h.ids("A1"),
	})
	if err != nil {
		t.Fatalf("counter sell: %v", err)
	}
	code := sold.Tickets[0].Code
	if code == "" {
		t.Fatal("counter ticket has no code")
	}

	status, _, body := h.call(http.MethodGet, "/api/v1/staff/tickets/"+code+"/order", staff, nil)
	if status != http.StatusOK {
		t.Fatalf("resolve: HTTP %d %v", status, body["message"])
	}
	if got := dataMap(body)["id"]; got != sold.ID {
		t.Fatalf("resolved order = %v, want %v", got, sold.ID)
	}

	if status, _, _ := h.call(http.MethodGet, "/api/v1/staff/tickets/"+code+"/order", customer, nil); status != http.StatusForbidden {
		t.Fatalf("customer resolve: HTTP %d, want 403", status)
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/staff/tickets/NOPE-NOT-A-CODE/order", staff, nil); status != http.StatusNotFound {
		t.Fatalf("unknown code: HTTP %d, want 404", status)
	}
}

// First collect stamps; repeats return the same stamp.
func TestHTTP_OrderCollectTickets(t *testing.T) {
	h := newHTTPEnv(t)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	sold, err := h.svc.CounterSell(h.ctx, dto.CounterSellRequest{
		ShowID: h.showID, SeatIDs: h.ids("A1"),
	})
	if err != nil {
		t.Fatalf("counter sell: %v", err)
	}

	status, _, body := h.call(http.MethodPost, "/api/v1/staff/orders/"+sold.ID+"/collect", staff, nil)
	if status != http.StatusOK {
		t.Fatalf("collect: HTTP %d %v", status, body["message"])
	}
	first := dataMap(body)["collected_at"]
	if first == nil || first == "" {
		t.Fatalf("collect did not stamp: %v", dataMap(body))
	}

	status, _, body = h.call(http.MethodPost, "/api/v1/staff/orders/"+sold.ID+"/collect", staff, nil)
	if status != http.StatusOK {
		t.Fatalf("re-collect: HTTP %d", status)
	}
	if dataMap(body)["collected_at"] != first {
		t.Fatal("re-collect moved the stamp")
	}

	if status, _, _ := h.call(http.MethodPost, "/api/v1/staff/orders/"+sold.ID+"/collect", customer, nil); status != http.StatusForbidden {
		t.Fatalf("customer collect: HTTP %d, want 403", status)
	}
	if status, _, _ := h.call(http.MethodPost, "/api/v1/staff/orders/00000000-0000-0000-0000-000000000000/collect", staff, nil); status != http.StatusNotFound {
		t.Fatalf("collect unknown: HTTP %d, want 404", status)
	}
}
