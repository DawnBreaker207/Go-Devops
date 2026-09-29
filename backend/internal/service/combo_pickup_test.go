package service_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

func comboSvc(t *testing.T, e *env) service.ComboService {
	t.Helper()
	return service.NewComboService(e.db, repository.NewComboRepository(e.db),
		repository.NewComboOrderRepository(e.db), repository.NewBookingRepository(e.db))
}

func seedCombo(t *testing.T, e *env, name string, price int64) string {
	t.Helper()
	c := &models.Combo{Name: name, Price: price, Active: true}
	e.must(e.db.Create(c).Error)
	return c.ID
}

func seedOnlineBooking(t *testing.T, e *env) (bookingID, ticketCode string) {
	t.Helper()
	b := &models.Booking{
		UserID: e.users[0], ShowtimeID: e.showID, Status: models.BookingPending,
		TotalAmount: priceStandard,
	}
	e.must(e.db.Create(b).Error)
	payID := "11111111-1111-1111-1111-111111111111"
	e.must(e.db.Exec(`INSERT INTO payments (id, booking_id, provider, txn_ref, amount, status, paid_amount, paid_at)
		VALUES (?, ?, 'mock', 'PICKUP-TXN', ?, 'paid', ?, NOW())`, payID, b.ID, priceStandard, priceStandard).Error)
	e.must(e.db.Exec(`UPDATE bookings SET status = 'confirmed', paid_at = NOW(), payment_id = ?
		WHERE id = ?`, payID, b.ID).Error)
	ticket := &models.Ticket{
		BookingID: b.ID, ShowtimeSeatID: e.seat["A1"],
		Price: priceStandard, Code: "PICKUP1", Status: models.TicketIssued,
	}
	e.must(e.db.Create(ticket).Error)
	return b.ID, ticket.Code
}

// Online pre-order sits on the board; collecting flips it once.
func TestCombo_PickupBoardAndCollect(t *testing.T) {
	e := newEnv(t)
	svc := comboSvc(t, e)
	popcorn := seedCombo(t, e, "Bap pickup", 59000)
	bookingID, ticketCode := seedOnlineBooking(t, e)

	ord, err := svc.CreateOrder(e.ctx, e.users[0], dto.CreateComboOrderRequest{
		BookingID: bookingID,
		Items:     []dto.ComboOrderItemRequest{{ComboID: popcorn, Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("create online combo order: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	rows, err := svc.PendingPickups(e.ctx, today, "")
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(rows) != 1 || rows[0].OrderID != ord.ID {
		t.Fatalf("pending = %+v, want the booked order", rows)
	}
	if rows[0].MovieTitle == "" || rows[0].BookingID != bookingID || len(rows[0].Items) != 1 {
		t.Fatalf("pickup row missing context: %+v", rows[0])
	}

	// Ticket-code search finds it; a far-future day does not.
	rows, err = svc.PendingPickups(e.ctx, today, ticketCode)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ticket search = %+v, %v", rows, err)
	}
	rows, err = svc.PendingPickups(e.ctx, "2036-01-01", "")
	if err != nil || len(rows) != 0 {
		t.Fatalf("future day = %+v, %v", rows, err)
	}

	got, err := svc.CollectOrder(e.ctx, ord.ID)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if got.Status != models.ComboOrderCollected {
		t.Fatalf("status = %v", got.Status)
	}
	if _, err := svc.CollectOrder(e.ctx, ord.ID); err == nil {
		t.Fatal("second collect succeeded, want not-found")
	}
	rows, err = svc.PendingPickups(e.ctx, today, "")
	if err != nil || len(rows) != 0 {
		t.Fatalf("pending after collect = %+v, %v", rows, err)
	}
}

// Booking-less online orders have no showtime to anchor: always due.
func TestCombo_PickupListsBookingLessOrders(t *testing.T) {
	e := newEnv(t)
	svc := comboSvc(t, e)
	popcorn := seedCombo(t, e, "Bap solo", 45000)

	ord, err := svc.CreateOrder(e.ctx, e.users[1], dto.CreateComboOrderRequest{
		Items: []dto.ComboOrderItemRequest{{ComboID: popcorn, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rows, err := svc.PendingPickups(e.ctx, "2036-01-01", "")
	if err != nil || len(rows) != 1 || rows[0].OrderID != ord.ID {
		t.Fatalf("booking-less pending = %+v, %v", rows, err)
	}
}

// Counter orders skip pickup; customers cannot collect.
func TestHTTP_ComboPickupRoles(t *testing.T) {
	h := newHTTPEnv(t)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	if status, _, _ := h.call(http.MethodGet, "/api/v1/staff/combo-orders/pending", customer, nil); status != http.StatusForbidden {
		t.Fatalf("customer pending board: HTTP %d, want 403", status)
	}
	if status, _, _ := h.call(http.MethodPost, "/api/v1/staff/combo-orders/00000000-0000-0000-0000-000000000000/collect", customer, nil); status != http.StatusForbidden {
		t.Fatalf("customer collect: HTTP %d, want 403", status)
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/staff/combo-orders/pending", staff, nil); status != http.StatusOK {
		t.Fatalf("staff pending board: HTTP %d", status)
	}
	if status, _, _ := h.call(http.MethodPost, "/api/v1/staff/combo-orders/00000000-0000-0000-0000-000000000000/collect", staff, nil); status != http.StatusNotFound {
		t.Fatalf("collect unknown: HTTP %d, want 404", status)
	}
}
