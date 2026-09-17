package service_test

import (
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

func (e *env) createTier(discountPercent float64) *models.MembershipTier {
	e.t.Helper()
	tier := &models.MembershipTier{Name: "Gold", Price: 499000, DurationDays: 365, DiscountPercent: discountPercent, Active: true}
	e.must(e.db.Create(tier).Error)
	return tier
}

func (e *env) grantMembership(userID string, tier *models.MembershipTier) {
	e.t.Helper()
	m := &models.UserMembership{UserID: userID, TierID: tier.ID, StartedAt: time.Now(), ExpiresAt: time.Now().Add(365 * 24 * time.Hour), Status: models.MembershipActive}
	e.must(e.db.Create(m).Error)
}

func (e *env) createCombo(price int64, memberPrice *int64) *models.Combo {
	e.t.Helper()
	c := &models.Combo{Name: "Combo bap nuoc", Price: price, MemberPrice: memberPrice, Active: true}
	e.must(e.db.Create(c).Error)
	return c
}

func (e *env) createVoucher(mutate func(v *models.Voucher)) *models.Voucher {
	e.t.Helper()
	adminID := e.users[0]
	v := &models.Voucher{
		Code: "TESTCODE", DiscountType: models.VoucherDiscountPercentage, DiscountValue: 10,
		MinOrderAmount: 0, StartsAt: time.Now().Add(-time.Hour), EndsAt: time.Now().Add(time.Hour),
		MaxUsage: 10, MaxUsagePerUser: 1, ApplyScope: models.VoucherScopeAll,
		CreatedByAdminID: &adminID, Status: models.VoucherActive,
	}
	if mutate != nil {
		mutate(v)
	}
	e.must(e.db.Create(v).Error)
	return v
}

// TestHold_MembershipDiscountsEverySeatPrice: an active membership's discount
// is baked into each held seat's price, same as the price itself.
func TestHold_MembershipDiscountsEverySeatPrice(t *testing.T) {
	e := newEnv(t)
	tier := e.createTier(10) // 10%
	e.grantMembership(e.users[0], tier)

	h := e.mustHold(e.users[0], "A1")
	want := priceStandard - priceStandard/10
	if h.Seats[0].Price != int64(want) {
		t.Fatalf("seat price = %d, want %d", h.Seats[0].Price, want)
	}
	if h.TotalAmount != int64(want) {
		t.Fatalf("total = %d, want %d", h.TotalAmount, want)
	}
}

// TestHold_ComboAttachesAndIsPriced: a combo requested at hold time is priced
// (member price when the user has an active membership) and added to the total.
func TestHold_ComboAttachesAndIsPriced(t *testing.T) {
	e := newEnv(t)
	memberPrice := int64(79000)
	combo := e.createCombo(89000, &memberPrice)
	tier := e.createTier(0.01) // negligible seat discount, isolates the combo price
	e.grantMembership(e.users[0], tier)

	res, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{
		ShowID: e.showID, SeatIDs: e.ids("A1"),
		Combos: []dto.ComboItem{{ComboID: combo.ID, Quantity: 2}},
	})
	e.must(err)
	if len(res.Combos) != 1 || res.Combos[0].Quantity != 2 || res.Combos[0].Price != memberPrice {
		t.Fatalf("combos = %+v, want 1 line qty=2 price=%d", res.Combos, memberPrice)
	}
	if res.TotalAmount <= priceStandard {
		t.Fatalf("total %d should include combo total on top of the seat price", res.TotalAmount)
	}
}

// TestHold_VoucherDiscountsTotalAndReservesUsage: a valid voucher discounts
// the booking total, and consumes one unit of usage_count + one redemption row.
func TestHold_VoucherDiscountsTotalAndReservesUsage(t *testing.T) {
	e := newEnv(t)
	v := e.createVoucher(nil) // 10% off, all scope

	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("A1"), VoucherCode: "testcode"})
	e.must(err)
	wantDiscount := int64(priceStandard) / 10
	if h.VoucherDiscountAmount != wantDiscount {
		t.Fatalf("voucher discount = %d, want %d", h.VoucherDiscountAmount, wantDiscount)
	}
	if h.TotalAmount != int64(priceStandard)-wantDiscount {
		t.Fatalf("total = %d, want %d", h.TotalAmount, int64(priceStandard)-wantDiscount)
	}

	var usage int
	e.must(e.db.Raw(`SELECT usage_count FROM vouchers WHERE id = ?`, v.ID).Scan(&usage).Error)
	if usage != 1 {
		t.Fatalf("voucher usage_count = %d, want 1", usage)
	}
	var redemptions int64
	e.must(e.db.Model(&models.VoucherRedemption{}).Where("voucher_id = ? AND booking_id = ?", v.ID, h.BookingID).Count(&redemptions).Error)
	if redemptions != 1 {
		t.Fatalf("redemptions = %d, want 1", redemptions)
	}
}

// TestCancel_ReleasesVoucherUsage: canceling a pending hold that used a
// voucher gives the usage slot back and removes the redemption row (Phần 1.1
// "Refund kỹ thuật phải rollback voucher").
func TestCancel_ReleasesVoucherUsage(t *testing.T) {
	e := newEnv(t)
	v := e.createVoucher(nil)

	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("A1"), VoucherCode: "TESTCODE"})
	e.must(err)

	_, err = e.svc.Cancel(e.ctx, e.users[0], h.BookingID)
	e.must(err)

	var usage int
	e.must(e.db.Raw(`SELECT usage_count FROM vouchers WHERE id = ?`, v.ID).Scan(&usage).Error)
	if usage != 0 {
		t.Fatalf("voucher usage_count after cancel = %d, want 0", usage)
	}
	var redemptions int64
	e.must(e.db.Model(&models.VoucherRedemption{}).Where("booking_id = ?", h.BookingID).Count(&redemptions).Error)
	if redemptions != 0 {
		t.Fatalf("redemptions after cancel = %d, want 0", redemptions)
	}
}

// TestHold_VoucherInvalidCodeIsGeneric: an unknown code is rejected the same
// way an expired/exhausted one would be, per the "chống dò mã" requirement.
func TestHold_VoucherInvalidCodeIsGeneric(t *testing.T) {
	e := newEnv(t)
	_, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids("A1"), VoucherCode: "NOPE"})
	if err == nil {
		t.Fatal("want an error for an unknown voucher code")
	}
}

// TestConfirm_WithComboAndVoucher_StaysConsistent: the full hold -> pay ->
// confirm flow still balances seats + combos - voucher against total_amount
// (booking_payment.go finalizeTx's invariant check).
func TestConfirm_WithComboAndVoucher_StaysConsistent(t *testing.T) {
	e := newEnv(t)
	combo := e.createCombo(50000, nil)
	e.createVoucher(nil)

	h, err := e.svc.Hold(e.ctx, e.users[0], dto.HoldRequest{
		ShowID: e.showID, SeatIDs: e.ids("A1"),
		Combos: []dto.ComboItem{{ComboID: combo.ID, Quantity: 1}}, VoucherCode: "TESTCODE",
	})
	e.must(err)

	b := e.payAndNotify(e.users[0], h.BookingID)
	if b.Status != models.BookingConfirmed {
		t.Fatalf("status = %s, want confirmed", b.Status)
	}
}

// TestHold_PricingRuleAdjustsSeatPrice: an always-on +20% rule raises every
// seat price before the membership discount is taken off it.
func TestHold_PricingRuleAdjustsSeatPrice(t *testing.T) {
	e := newEnv(t)
	rule := &models.PricingRule{Name: "Weekend surcharge", AdjustmentType: models.PricingAdjustPercent, AdjustmentValue: 20, Active: true}
	e.must(e.db.Create(rule).Error)

	h := e.mustHold(e.users[0], "A1")
	want := priceStandard + priceStandard/5 // +20%
	if h.Seats[0].Price != int64(want) {
		t.Fatalf("price = %d, want %d", h.Seats[0].Price, want)
	}
}
