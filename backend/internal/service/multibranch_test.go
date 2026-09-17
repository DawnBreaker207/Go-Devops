package service_test

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

// secondBranchShowtime creates a second branch, a hall in it, and one open
// showtime — for tests that need two branches to cross-check scoping.
func (e *env) secondBranchShowtime() (branchID, showID string) {
	e.t.Helper()
	branch := &models.Branch{Name: "Chi nhanh 2", Active: true}
	e.must(e.db.Create(branch).Error)

	hall, err := e.halls.Create(e.ctx, dto.HallRequest{
		BranchID: branch.ID, Name: "Hall Branch2", Rows: 2, SeatsPerRow: 5,
		Prices: fullPrices(),
	})
	e.must(err)

	st, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: hall.ID, StartAt: time.Now().Add(3 * time.Hour)})
	e.must(err)
	return branch.ID, st.ID
}

// TestHold_BranchScopedVoucherRejectedAcrossBranches: a voucher scoped to
// branch A can not discount a booking whose showtime is in branch B.
func TestHold_BranchScopedVoucherRejectedAcrossBranches(t *testing.T) {
	e := newEnv(t)
	otherBranchID, otherShowID := e.secondBranchShowtime()
	seats := e.seatsOf(otherShowID)
	var seatID string
	for id := range seats {
		seatID = id
		break
	}

	// Voucher scoped to the DEFAULT branch (not the one the showtime is in).
	var defaultBranchID string
	e.must(e.db.Raw(`SELECT branch_id FROM halls WHERE id = ?`, e.hallID).Scan(&defaultBranchID).Error)
	if defaultBranchID == otherBranchID {
		t.Fatal("test setup: the two branches must differ")
	}
	e.createVoucher(func(v *models.Voucher) { v.BranchID = &defaultBranchID })

	_, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: otherShowID, SeatIDs: []string{seatID}, VoucherCode: "TESTCODE"})
	if err == nil {
		t.Fatal("want an error: voucher is scoped to a different branch")
	}
}

// TestRedeem_WrongBranchRejected: a staff account scoped to branch A can not
// check a ticket in for a showtime in branch B.
func TestRedeem_WrongBranchRejected(t *testing.T) {
	e := newEnv(t)
	otherBranchID, otherShowID := e.secondBranchShowtime()

	staff := e.newUser(models.RoleStaff, "gate-staff@test.local", "x")
	e.must(e.db.Exec(`UPDATE users SET branch_id = ? WHERE id = ?`, otherBranchID, staff.ID).Error)

	// Book and confirm a ticket on the DEFAULT branch's showtime (e.showID).
	bookingID := e.confirmed(e.users[0], "A1")
	order, err := e.svc.Order(e.ctx, e.users[0], bookingID)
	e.must(err)

	res, err := e.svc.Redeem(e.ctx, order.Tickets[0].Code, e.showID, staff.ID)
	e.must(err)
	if res.Status != models.RedeemWrongBranch {
		t.Fatalf("status = %s, want wrong_branch", res.Status)
	}
	_ = otherShowID
}

// TestHold_ComboOutOfStockRejected: a combo capped at 0 stock in the
// booking's branch can not be added to a hold; canceling a successful
// reservation gives the stock back.
func TestHold_ComboOutOfStockRejected(t *testing.T) {
	e := newEnv(t)
	combo := e.createCombo(50000, nil)
	comboRepo := repository.NewComboRepository(e.db)

	var branchID string
	e.must(e.db.Raw(`SELECT branch_id FROM halls WHERE id = ?`, e.hallID).Scan(&branchID).Error)
	e.must(e.db.Transaction(func(tx *gorm.DB) error { return comboRepo.SetStock(tx, branchID, combo.ID, 1) }))

	// First reservation succeeds (1 in stock).
	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{
		ShowID: e.showID, SeatIDs: e.ids("A1"), Combos: []dto.ComboItem{{ComboID: combo.ID, Quantity: 1}},
	})
	e.must(err)

	// A second user trying to reserve more than what's left is refused.
	_, err = e.svc.Hold(e.ctx, e.users[1], dto.HoldRequest{
		ShowID: e.showID, SeatIDs: e.ids("A2"), Combos: []dto.ComboItem{{ComboID: combo.ID, Quantity: 1}},
	})
	if err == nil {
		t.Fatal("want an error: combo is out of stock at this branch")
	}

	// Canceling the first booking gives the unit back.
	_, err = e.svc.Cancel(e.ctx, e.users[0], h.BookingID)
	e.must(err)
	var stock int
	e.must(e.db.Raw(`SELECT stock_quantity FROM combo_branch_stock WHERE branch_id = ? AND combo_id = ?`, branchID, combo.ID).Scan(&stock).Error)
	if stock != 1 {
		t.Fatalf("stock after cancel = %d, want 1", stock)
	}
}
