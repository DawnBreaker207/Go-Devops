package service_test

import (
	"net/http"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// POST /admin/showtimes/:id/cancel is the ONLY way to stop a showtime that has
// already sold tickets — DELETE is refused with 409 the moment a booking exists.
// It moves real money, and until now it had no test at all, which mattered less
// while nothing in the app could reach it. The operator screen can now.
func TestHTTP_CancelShowtimeRefundsAndVoidsTickets(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	bookingID := h.confirmed(h.users[0], "A1", "A2")

	// DELETE must refuse first: that refusal is the reason cancel exists.
	status, _, body := h.call(http.MethodDelete, "/api/v1/admin/showtimes/"+h.showID, admin, nil)
	if status != http.StatusConflict {
		t.Fatalf("delete a sold showtime: HTTP %d %v, want 409", status, body["message"])
	}

	status, _, body = h.call(http.MethodPost, "/api/v1/admin/showtimes/"+h.showID+"/cancel", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("cancel: HTTP %d %v", status, body["message"])
	}
	result := dataMap(body)
	if result["status"] != models.ShowtimeCancelled {
		t.Fatalf("showtime status = %v, want %q", result["status"], models.ShowtimeCancelled)
	}
	if affected, _ := result["bookings_affected"].(float64); affected != 1 {
		t.Fatalf("bookings_affected = %v, want 1", result["bookings_affected"])
	}

	// The booking is refunded with the reason that says WHY, and its tickets are
	// void — an issued ticket for a cancelled showtime would still scan at the door.
	b := h.wantStatus(bookingID, models.BookingRefunded)
	if b.StatusReason == nil || *b.StatusReason != models.ReasonShowtimeCancelled {
		t.Fatalf("status_reason = %v, want %q", b.StatusReason, models.ReasonShowtimeCancelled)
	}
	if n := h.count(`SELECT count(*) FROM tickets WHERE booking_id = ? AND status <> ?`,
		bookingID, models.TicketVoid); n != 0 {
		t.Fatalf("%d ticket(s) left un-voided on a cancelled showtime", n)
	}

	// Sold seats are deliberately NOT released. The showtime is dead, so they can
	// never be resold, and leaving them `sold` keeps the record of who bought what.
	// Only PENDING holds are released (see CancelShowtime). Asserted so a future
	// change to release them is a conscious decision rather than a silent one.
	if n := h.count(`SELECT count(*) FROM showtime_seats WHERE showtime_id = ? AND status = ?`,
		h.showID, models.SeatStatusSold); n != 2 {
		t.Fatalf("sold seats = %d, want the 2 to stay sold on a cancelled showtime", n)
	}

	// What must be true instead: the cancelled showtime takes no new bookings.
	status, _, body = h.call(http.MethodPost, "/api/v1/orders/hold", customer, map[string]any{
		"show_id": h.showID, "seat_ids": h.ids("B1"),
	})
	if status == http.StatusOK || status == http.StatusCreated {
		t.Fatalf("a cancelled showtime accepted a new hold (HTTP %d %v)", status, body["message"])
	}

	// Cancelling twice is a conflict, not a second round of refunds.
	status, _, body = h.call(http.MethodPost, "/api/v1/admin/showtimes/"+h.showID+"/cancel", admin, nil)
	if status != http.StatusConflict {
		t.Fatalf("cancel twice: HTTP %d %v, want 409", status, body["message"])
	}
}

// Staff may cancel too (the route is admin+staff like the rest of /admin/showtimes),
// but a customer must never reach an endpoint that refunds other people's orders.
func TestHTTP_CancelShowtimeScope(t *testing.T) {
	h := newHTTPEnv(t)
	customer, _ := h.login(models.RoleCustomer)

	status, _, body := h.call(http.MethodPost, "/api/v1/admin/showtimes/"+h.showID+"/cancel", customer, nil)
	if status != http.StatusForbidden {
		t.Fatalf("customer cancels a showtime: HTTP %d %v, want 403", status, body["message"])
	}

	status, _, _ = h.call(http.MethodPost, "/api/v1/admin/showtimes/"+h.showID+"/cancel", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous cancels a showtime: HTTP %d, want 401", status)
	}
}
