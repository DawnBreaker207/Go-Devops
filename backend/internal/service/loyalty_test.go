package service_test

import (
	"testing"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
)

// TestConfirm_EarnsPoints: a confirmed online booking credits floor(total/1000) points.
func TestConfirm_EarnsPoints(t *testing.T) {
	e := newEnv(t)
	bookingID := e.confirmed(e.users[0], "A1")
	b := e.booking(bookingID)

	loyalty := repository.NewLoyaltyRepository(e.db)
	balance, err := loyalty.Balance(e.ctx, e.users[0])
	e.must(err)
	want := b.TotalAmount / 1000
	if balance != want {
		t.Fatalf("balance = %d, want %d", balance, want)
	}
}

// TestEarnForBooking_MilestoneIssuesUsableVoucher: crossing the 10-ticket
// milestone in one call issues a voucher usable only by the earning user.
// EarnForBooking is exercised directly (rather than through 10 real seat
// sales) because the test hall only has 9 sellable seats.
func TestEarnForBooking_MilestoneIssuesUsableVoucher(t *testing.T) {
	e := newEnv(t)
	loyaltyRepo := repository.NewLoyaltyRepository(e.db)
	ledgerRepo := repository.NewLedgerRepository(e.db)
	voucherRepo := repository.NewVoucherRepository(e.db)
	svc := service.NewLoyaltyService(e.db, loyaltyRepo, voucherRepo, ledgerRepo)

	// One real booking so a valid booking_id exists for the FK on
	// point_transactions.reference_booking_id, then top up the remaining
	// 9 tickets synthetically to cross the milestone without needing 10
	// real seats.
	bookingID := e.confirmed(e.users[0], "A1")
	e.must(e.db.Transaction(func(tx *gorm.DB) error {
		return svc.EarnForBooking(e.ctx, tx, e.users[0], priceStandard, 9, bookingID)
	}))

	var code string
	e.must(e.db.Raw(`SELECT code FROM vouchers WHERE assigned_user_id = ? AND is_system_issued = true LIMIT 1`, e.users[0]).Scan(&code).Error)
	if code == "" {
		t.Fatal("expected a milestone voucher to have been issued")
	}

	// The earning user can use it...
	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("B1"), VoucherCode: code})
	e.must(err)
	if h.VoucherDiscountAmount <= 0 {
		t.Fatalf("voucher discount = %d, want > 0", h.VoucherDiscountAmount)
	}

	// ...but a different user can not (assigned_user_id lock).
	_, err = e.svc.Hold(e.ctx, e.users[1], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("B2"), VoucherCode: code})
	if err == nil {
		t.Fatal("want an error: voucher is locked to a different user")
	}
}

// TestRedeem_InsufficientPointsIsRejected: the CAS spend refuses a redeem
// past the balance, never going negative.
func TestRedeem_InsufficientPointsIsRejected(t *testing.T) {
	e := newEnv(t)
	loyaltyRepo := repository.NewLoyaltyRepository(e.db)
	ledgerRepo := repository.NewLedgerRepository(e.db)
	voucherRepo := repository.NewVoucherRepository(e.db)
	svc := service.NewLoyaltyService(e.db, loyaltyRepo, voucherRepo, ledgerRepo)

	reward := &models.Reward{Type: models.RewardGift, Name: "Ly giu nhiet", PointsCost: 1000, Active: true}
	e.must(e.db.Create(reward).Error)

	_, err := svc.Redeem(e.ctx, e.users[0], reward.ID)
	if err == nil {
		t.Fatal("want an error: user has 0 points")
	}
}
