package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

const maxTxAttempts = 4

const TicketEmailQueue = "email.tickets"

const (
	sourceCustomer  = "customer"
	sourceReconcile = "reconcile"
	sourceSweep     = "sweep"
)

type JobPublisher interface {
	Publish(ctx context.Context, queueName string, body []byte) error
}

type SweepResult struct {
	ReleasedSeats      int
	ExpiredBookings    int
	FinalizedPaid      int
	SettledRefunds     int
	ReconciledPayments int
	AbandonedPayments  int
	LateCaptures       int
}

func (r SweepResult) Total() int {
	return r.ReleasedSeats + r.ExpiredBookings + r.FinalizedPaid + r.SettledRefunds +
		r.ReconciledPayments + r.AbandonedPayments + r.LateCaptures
}

// BookingService: payment goes through payment.Provider only, so the mock and real
// gateways take exactly the same path.
type BookingService interface {
	Hold(ctx context.Context, userID string, req dto.HoldRequest) (*dto.HoldResponse, error)
	Pay(ctx context.Context, userID, bookingID string, req dto.PayRequest) (*dto.PayResponse, error)
	HandleNotification(ctx context.Context, provider payment.Provider, r *http.Request) payment.AckStatus
	HandleReturn(ctx context.Context, provider payment.Provider, r *http.Request) (*dto.PaymentReturnResponse, error)
	Confirm(ctx context.Context, userID, bookingID string) (*dto.OrderDetailResponse, error)
	Status(ctx context.Context, userID, bookingID string) (*dto.OrderStatusResponse, error)
	Order(ctx context.Context, userID, bookingID string) (*dto.OrderDetailResponse, error)
	AdminOrder(ctx context.Context, bookingID string) (*dto.OrderDetailResponse, error)
	Cancel(ctx context.Context, userID, bookingID string) (*dto.OrderStatusResponse, error)
	List(ctx context.Context, userID string, q dto.PageQuery) ([]dto.OrderStatusResponse, int64, error)
	// staffUserID's branch (if any) gates the check-in (Phần 4); an unscoped
	// staff account checks in at any branch.
	Redeem(ctx context.Context, ticketRef, showtimeID, staffUserID string) (*dto.RedeemResponse, error)
	CounterSell(ctx context.Context, req dto.CounterSellRequest) (*dto.OrderDetailResponse, error)
	SweepExpired(ctx context.Context, limit int) (SweepResult, error)
}

// BookingOptions wires the booking service.
type BookingOptions struct {
	DB            *gorm.DB
	Repo          repository.BookingRepository
	Payments      repository.PaymentRepository
	Vouchers      *repository.VoucherRepository
	Memberships   *repository.MembershipRepository
	Combos        *repository.ComboRepository
	Loyalty       LoyaltyService
	Ledger        *repository.LedgerRepository
	Waitlist      *repository.WaitlistRepository
	PricingRules  *repository.PricingRuleRepository
	Branches      *repository.BranchRepository
	Queue         QueueService
	Providers     *payment.Registry
	PublicBaseURL string
	HoldTTL       time.Duration
	MaxSeats      int
	// Publisher may be nil: the sendTicketEmails cron picks unsent emails up.
	Publisher JobPublisher
	// Hub may be nil (no realtime seat updates).
	Hub *sse.Hub
	// LateCaptureWindow: how long given-up attempts are rechecked for late money; 0 means 24h.
	LateCaptureWindow time.Duration
	// Check-in window around the showtime start; both zero means 30 and 20 minutes.
	CheckinOpenBefore time.Duration
	CheckinCloseAfter time.Duration
}

type bookingService struct {
	db                *gorm.DB
	repo              repository.BookingRepository
	payments          repository.PaymentRepository
	vouchers          *repository.VoucherRepository
	memberships       *repository.MembershipRepository
	combos            *repository.ComboRepository
	loyalty           LoyaltyService
	ledger            *repository.LedgerRepository
	waitlist          *repository.WaitlistRepository
	pricingRules      *repository.PricingRuleRepository
	branches          *repository.BranchRepository
	queue             QueueService
	providers         *payment.Registry
	publicBaseURL     string
	holdTTL           time.Duration
	maxSeats          int
	publisher         JobPublisher
	hub               *sse.Hub
	lateCaptureWindow time.Duration
	checkinOpenBefore time.Duration
	checkinCloseAfter time.Duration
}

func NewBookingService(opts BookingOptions) BookingService {
	if opts.LateCaptureWindow <= 0 {
		opts.LateCaptureWindow = defaultLateCaptureWindow
	}
	if opts.CheckinOpenBefore == 0 && opts.CheckinCloseAfter == 0 {
		opts.CheckinOpenBefore, opts.CheckinCloseAfter = defaultCheckinOpenBefore, defaultCheckinCloseAfter
	}
	return &bookingService{
		db:                opts.DB,
		repo:              opts.Repo,
		payments:          opts.Payments,
		vouchers:          opts.Vouchers,
		memberships:       opts.Memberships,
		combos:            opts.Combos,
		loyalty:           opts.Loyalty,
		ledger:            opts.Ledger,
		waitlist:          opts.Waitlist,
		pricingRules:      opts.PricingRules,
		branches:          opts.Branches,
		queue:             opts.Queue,
		providers:         opts.Providers,
		publicBaseURL:     strings.TrimRight(opts.PublicBaseURL, "/"),
		holdTTL:           opts.HoldTTL,
		maxSeats:          opts.MaxSeats,
		publisher:         opts.Publisher,
		hub:               opts.Hub,
		lateCaptureWindow: opts.LateCaptureWindow,
		checkinOpenBefore: opts.CheckinOpenBefore,
		checkinCloseAfter: opts.CheckinCloseAfter,
	}
}

// inTx re-runs fn when the transaction lost a race; fn must reset what it captures on every attempt.
func (s *bookingService) inTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	var err error
	for attempt := 0; attempt < maxTxAttempts; attempt++ {
		err = s.db.WithContext(ctx).Transaction(fn)
		if err == nil || !apperrors.IsRetryable(err) {
			return err
		}
	}
	return err
}

// Hold locks seat rows in id order and takes expiry from the DB clock. A previous unpaid
// hold of the same user and show is replaced without extending its expiry.
func (s *bookingService) Hold(ctx context.Context, userID string, req dto.HoldRequest) (*dto.HoldResponse, error) {
	seatIDs := uniqueStrings(req.SeatIDs)
	if len(seatIDs) == 0 {
		return nil, apperrors.Validation("seat_ids must not be empty")
	}
	if len(seatIDs) > s.maxSeats {
		return nil, apperrors.ErrSeatLimitExceeded.WithDetails(map[string]string{"max": strconv.Itoa(s.maxSeats)})
	}
	key := strings.TrimSpace(req.IdempotencyKey)

	// Virtual queue gate (Phần 2.3): sits in FRONT of the hold transaction,
	// never replaces its Postgres locking. Only consulted when an admin
	// flagged this showtime queue_enabled; Redis unreachable fails open
	// (queueService already no-ops on a nil cache).
	if s.queue != nil {
		enabled, err := s.repo.ShowtimeQueueEnabled(ctx, req.ShowID)
		if err != nil {
			return nil, err
		}
		if enabled {
			admitted, qerr := s.queue.IsAdmitted(ctx, req.ShowID, userID)
			if qerr != nil {
				// Fail-open (Phần 2.3): a broken queue must never block a
				// sale — skip the gate rather than surface the error.
				logger.Warn("virtual queue check failed; letting the hold through", logger.Err(qerr))
				admitted = true
			}
			if !admitted {
				return nil, apperrors.Conflict("this showtime has a virtual queue; join it first (POST /queue/join) and wait to be admitted")
			}
		}
	}

	var (
		result   *dto.HoldResponse
		released []string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		var err error
		result, released, err = s.holdTx(ctx, tx, userID, req.ShowID, seatIDs, key, req.Combos, req.VoucherCode)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.broadcast(req.ShowID, models.SeatStatusAvailable, released)
	held := make([]string, 0, len(result.Seats))
	for _, seat := range result.Seats {
		held = append(held, seat.ShowtimeSeatID)
	}
	s.broadcast(req.ShowID, models.SeatStatusHeld, held)
	return result, nil
}

func (s *bookingService) holdTx(ctx context.Context, tx *gorm.DB, userID, showID string, seatIDs []string, key string,
	comboItems []dto.ComboItem, voucherCode string) (*dto.HoldResponse, []string, error) {
	if err := s.repo.LockUserShow(ctx, tx, userID, showID); err != nil {
		return nil, nil, err
	}
	active, err := s.repo.UserActive(ctx, tx, userID)
	if err != nil {
		return nil, nil, err
	}
	if !active {
		return nil, nil, apperrors.ErrAccountLocked
	}
	now, err := s.repo.Now(ctx, tx)
	if err != nil {
		return nil, nil, err
	}
	showtime, err := s.repo.LockShowtime(ctx, tx, showID)
	if err != nil {
		return nil, nil, err
	}
	if showtime == nil {
		return nil, nil, apperrors.ErrShowtimeNotFound
	}
	if showtime.Status != models.ShowtimeOpen || !showtime.StartAt.After(now) {
		return nil, nil, apperrors.ErrShowtimeClosed
	}
	// A movie taken off the schedule sells nothing, whatever its showtimes say.
	showing, err := s.repo.MovieShowing(ctx, tx, showtime.MovieID)
	if err != nil {
		return nil, nil, err
	}
	if !showing {
		return nil, nil, apperrors.ErrShowtimeClosed
	}

	// A retry with the same key gets back the booking it created. The key is spent by that
	// request: reused by another user, show or seat set, or once its booking ended, it is refused.
	if key != "" {
		existing, err := s.repo.LockLatestByKey(ctx, tx, key)
		if err != nil {
			return nil, nil, err
		}
		if existing != nil {
			if existing.UserID != userID || existing.ShowtimeID != showID || existing.Status != models.BookingPending {
				return nil, nil, apperrors.ErrIdempotencyKeyReused
			}
			same, err := s.sameSeats(ctx, tx, existing.ID, seatIDs)
			if err != nil {
				return nil, nil, err
			}
			if !same {
				return nil, nil, apperrors.ErrIdempotencyKeyReused
			}
			if existing.ExpiresAt == nil || !existing.ExpiresAt.After(now) {
				return nil, nil, apperrors.ErrBookingExpired
			}
			res, err := s.holdResponse(ctx, tx, existing)
			return res, nil, err
		}
	}

	heldUntil := now.Add(s.holdTTL)
	ownVersions := map[string]int64{}
	replacedID := ""

	// One PENDING booking per user per show. The new hold replaces the old one,
	// keeping the old expiry so tabs can not extend a hold forever.
	old, err := s.repo.LockPendingByUserShow(ctx, tx, userID, showID)
	if err != nil {
		return nil, nil, err
	}
	if old != nil {
		stillValid := old.ExpiresAt != nil && old.ExpiresAt.After(now)
		// Money may already be on its way for the old booking: replacing it would
		// silently drop a paid order.
		if old.PaidAt != nil {
			return nil, nil, apperrors.ErrPaymentInProgress
		}
		if stillValid {
			open, err := s.payments.HasOpen(ctx, tx, old.ID)
			if err != nil {
				return nil, nil, err
			}
			if open {
				return nil, nil, apperrors.ErrPaymentInProgress
			}
		}
		oldSeats, err := s.repo.BookingSeats(ctx, tx, old.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, bs := range oldSeats {
			ownVersions[bs.ShowtimeSeatID] = bs.HoldVersion
		}
		reason := models.ReasonHoldExpired
		if stillValid {
			reason = models.ReasonReplaced
			heldUntil = *old.ExpiresAt
		}
		n, err := s.repo.ExpireBooking(ctx, tx, old.ID, reason)
		if err != nil {
			return nil, nil, err
		}
		if n == 0 {
			return nil, nil, apperrors.Internal("replaced booking changed under lock")
		}
		if s.vouchers != nil && old.VoucherID != nil {
			if err := s.vouchers.ReleaseUsage(ctx, tx, old.ID); err != nil {
				return nil, nil, err
			}
		}
		if err := s.releaseComboStock(ctx, tx, old.ShowtimeID, old.ID); err != nil {
			return nil, nil, err
		}
		if err := s.audit(ctx, tx, "orders.expire", "booking", old.ID, old.ID,
			map[string]any{"status": models.BookingExpired, "reason": reason, "source": "new_hold"}); err != nil {
			return nil, nil, err
		}
		replacedID = old.ID
	}

	lockIDs := slices.Clone(seatIDs)
	for id := range ownVersions {
		if !slices.Contains(seatIDs, id) {
			lockIDs = append(lockIDs, id)
		}
	}
	locked, err := s.repo.LockSeats(ctx, tx, showID, lockIDs)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]models.ShowtimeSeat, len(locked))
	for _, seat := range locked {
		byID[seat.ID] = seat
	}

	physIDs := make([]string, 0, len(seatIDs))
	for _, id := range seatIDs {
		seat, ok := byID[id]
		if !ok {
			return nil, nil, apperrors.Validation("seat does not belong to this showtime").
				WithDetails(map[string]string{"seat_id": id})
		}
		physIDs = append(physIDs, seat.SeatID)
	}
	phys, err := s.repo.SeatsByIDs(ctx, tx, physIDs)
	if err != nil {
		return nil, nil, err
	}
	prices, err := s.repo.PricesByHall(ctx, tx, showtime.HallID)
	if err != nil {
		return nil, nil, err
	}

	// All or nothing: report every seat that failed.
	var taken, gaps []string
	for _, id := range seatIDs {
		seat := byID[id]
		meta := phys[seat.SeatID]
		label := dto.SeatLabel(meta.RowLabel, meta.ColNumber)
		switch {
		case meta.IsGap:
			gaps = append(gaps, label)
		case !holdable(seat, userID, ownVersions, now):
			taken = append(taken, label)
		}
	}
	if len(gaps) > 0 {
		return nil, nil, apperrors.ErrSeatNotSellable.WithDetails(map[string]string{"seats": strings.Join(gaps, ",")})
	}
	if len(taken) > 0 {
		return nil, nil, apperrors.ErrSeatTaken.WithDetails(map[string]string{"seats": strings.Join(taken, ",")})
	}

	var released []string
	for id, version := range ownVersions {
		if slices.Contains(seatIDs, id) {
			continue
		}
		n, err := s.repo.ReleaseHeldSeat(ctx, tx, id, userID, version)
		if err != nil {
			return nil, nil, err
		}
		if n > 0 {
			released = append(released, id)
		}
	}

	// An active membership discounts every seat price, baked in at hold time
	// just like the seat price itself (ADVANCED_FEATURES_DISCUSSION.md Phần 1.2).
	var membershipID *string
	var membershipPercent float64
	var hasMembership bool
	if s.memberships != nil {
		membership, tier, err := s.memberships.ActiveForUser(ctx, tx, userID)
		if err != nil {
			return nil, nil, err
		}
		if membership != nil && tier != nil {
			membershipID = &membership.ID
			membershipPercent = tier.DiscountPercent
			hasMembership = true
		}
	}

	// Dynamic pricing (day-of-week/time-of-day rules) adjusts the sticker
	// price before the membership discount is taken off it — the rule
	// changes what the seat is worth right now, membership is a discount off
	// that (Product Backlog "giá động theo ngày/khung giờ").
	var pricingRules []models.PricingRule
	if s.pricingRules != nil {
		var err error
		pricingRules, err = s.pricingRules.ActiveForHold(tx)
		if err != nil {
			return nil, nil, err
		}
	}

	var total int64
	held := make([]dto.HeldSeat, 0, len(seatIDs))
	bookingSeats := make([]models.BookingSeat, 0, len(seatIDs))
	labels := make([]string, 0, len(seatIDs))
	for _, id := range seatIDs {
		meta := phys[byID[id].SeatID]
		price, ok := prices[meta.SeatType]
		if !ok || price <= 0 {
			return nil, nil, apperrors.ErrMissingHallPrice.WithDetails(map[string]string{"seat_type": meta.SeatType})
		}
		price = applyPricingRules(pricingRules, price, showtime.StartAt)
		price = applyMembershipDiscount(price, membershipPercent)
		version, err := s.repo.HoldSeat(ctx, tx, id, userID, heldUntil)
		if err != nil {
			return nil, nil, err
		}
		total += price
		label := dto.SeatLabel(meta.RowLabel, meta.ColNumber)
		labels = append(labels, label)
		bookingSeats = append(bookingSeats, models.BookingSeat{
			ShowtimeSeatID: id,
			SeatType:       meta.SeatType,
			Price:          price,
			HoldVersion:    version,
		})
		held = append(held, dto.HeldSeat{
			ShowtimeSeatID: id,
			Label:          label,
			RowLabel:       meta.RowLabel,
			ColNumber:      meta.ColNumber,
			SeatType:       meta.SeatType,
			Price:          price,
		})
	}
	seatTotal := total

	// Combos are priced (member price if an active membership applies) but not
	// yet inserted: booking_combos needs the booking's id, created below.
	var comboRows []models.BookingCombo
	var comboTotal int64
	if s.combos != nil && len(comboItems) > 0 {
		var branchID string
		if s.branches != nil {
			var err error
			branchID, err = s.branches.HallBranch(tx, showtime.HallID)
			if err != nil {
				return nil, nil, err
			}
		}
		var err error
		comboRows, comboTotal, err = s.priceCombos(ctx, tx, comboItems, hasMembership, branchID)
		if err != nil {
			return nil, nil, err
		}
		total += comboTotal
	}

	// Voucher is locked and validated now (advisory -> user -> showtime -> seats
	// -> voucher lock order, Phần 1.1), but usage_count/redemption are only
	// written after the booking row exists, so voucher validation never blocks
	// on a booking id that does not exist yet.
	var voucher *models.Voucher
	var voucherDiscount int64
	if s.vouchers != nil && strings.TrimSpace(voucherCode) != "" {
		var err error
		voucher, voucherDiscount, err = s.lockAndValidateVoucher(tx, voucherCode, userID, showtime.HallID, now, seatTotal, comboTotal)
		if err != nil {
			return nil, nil, err
		}
		total -= voucherDiscount
	}

	booking := &models.Booking{
		UserID:                userID,
		ShowtimeID:            showID,
		Status:                models.BookingPending,
		TotalAmount:           total,
		VoucherDiscountAmount: voucherDiscount,
		MembershipID:          membershipID,
		ExpiresAt:             &heldUntil,
	}
	if voucher != nil {
		booking.VoucherID = &voucher.ID
	}
	if key != "" {
		booking.IdempotencyKey = &key
	}
	// A unique violation here (second tab or same key racing) re-runs the
	// whole transaction, which then finds and replaces/returns the winner.
	if err := s.repo.Create(ctx, tx, booking); err != nil {
		return nil, nil, err
	}
	for i := range bookingSeats {
		bookingSeats[i].BookingID = booking.ID
	}
	if err := s.repo.CreateBookingSeats(ctx, tx, bookingSeats); err != nil {
		return nil, nil, err
	}
	comboResponses := make([]dto.BookingComboResponse, 0, len(comboRows))
	if len(comboRows) > 0 {
		for i := range comboRows {
			comboRows[i].BookingID = booking.ID
		}
		if err := s.combos.CreateBookingCombos(tx, comboRows); err != nil {
			return nil, nil, err
		}
		for _, c := range comboRows {
			comboResponses = append(comboResponses, dto.BookingComboResponse{ComboID: c.ComboID, Quantity: c.Quantity, Price: c.Price})
		}
	}
	if voucher != nil {
		if n, err := s.vouchers.IncrUsage(tx, voucher.ID); err != nil {
			return nil, nil, err
		} else if n == 0 {
			// Lost the race for the last unit of the campaign cap: fail the whole hold.
			return nil, nil, apperrors.ErrVoucherInvalid
		}
		if err := s.vouchers.CreateRedemption(tx, &models.VoucherRedemption{VoucherID: voucher.ID, UserID: userID, BookingID: booking.ID}); err != nil {
			return nil, nil, err
		}
	}

	after := map[string]any{"showtime_id": showID, "seats": labels, "total": total, "expires_at": heldUntil}
	if replacedID != "" {
		after["replaced_booking_id"] = replacedID
	}
	if voucher != nil {
		after["voucher_code"] = voucher.Code
		after["voucher_discount_amount"] = voucherDiscount
		if voucher.CreatedByAdminID != nil && *voucher.CreatedByAdminID == userID {
			// Phần 9.4: the voucher's own creator redeeming it is a
			// conflict-of-interest signal — logged, never blocked (an owner
			// reviews GET /admin/vouchers/conflicts).
			after["conflict_of_interest"] = true
		}
	}
	if err := s.audit(ctx, tx, "orders.hold", "booking", booking.ID, booking.ID, after); err != nil {
		return nil, nil, err
	}

	resp := &dto.HoldResponse{
		BookingID:             booking.ID,
		ShowtimeID:            showID,
		TotalAmount:           total,
		ExpiresAt:             heldUntil,
		Seats:                 held,
		Combos:                comboResponses,
		VoucherDiscountAmount: voucherDiscount,
		ReplacedBookingID:     replacedID,
	}
	if voucher != nil {
		resp.VoucherCode = voucher.Code
	}
	return resp, released, nil
}

// applyPricingRules stacks every matching rule additively off the original
// hall price `base` (not off a previous rule's result), so rule evaluation
// order never changes the outcome. `at` is the showtime's own start time
// (dynamic pricing is about the time slot being sold, not the moment of
// purchase).
func applyPricingRules(rules []models.PricingRule, base int64, at time.Time) int64 {
	price := base
	dow := int(at.Weekday())
	clock := at.Format("15:04:05")
	for _, r := range rules {
		if r.DayOfWeek != nil && *r.DayOfWeek != dow {
			continue
		}
		if r.StartsAt != nil && r.EndsAt != nil && !(clock >= *r.StartsAt && clock < *r.EndsAt) {
			continue
		}
		switch r.AdjustmentType {
		case models.PricingAdjustPercent:
			price += int64(math.Round(float64(base) * r.AdjustmentValue / 100))
		default:
			price += int64(r.AdjustmentValue)
		}
	}
	if price < 0 {
		return 0
	}
	return price
}

// applyMembershipDiscount rounds to the nearest VND, consistent across every
// seat of the booking.
func applyMembershipDiscount(price int64, percent float64) int64 {
	if percent <= 0 {
		return price
	}
	discounted := price - int64(math.Round(float64(price)*percent/100))
	if discounted < 0 {
		return 0
	}
	return discounted
}

// priceCombos validates every requested combo is active and prices it (member
// price when the booking's user has an active membership), without touching
// booking_combos yet — the caller sets BookingID once the booking exists.
func (s *bookingService) priceCombos(ctx context.Context, tx *gorm.DB, items []dto.ComboItem, hasMembership bool, branchID string) ([]models.BookingCombo, int64, error) {
	ids := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if seen[item.ComboID] {
			return nil, 0, apperrors.Validation("duplicate combo_id in request").WithDetails(map[string]string{"combo_id": item.ComboID})
		}
		seen[item.ComboID] = true
		ids = append(ids, item.ComboID)
	}
	combos, err := s.combos.FindByIDs(ctx, tx, ids)
	if err != nil {
		return nil, 0, err
	}
	rows := make([]models.BookingCombo, 0, len(items))
	var total int64
	for _, item := range items {
		combo, ok := combos[item.ComboID]
		if !ok || !combo.Active {
			return nil, 0, apperrors.ErrComboInactive.WithDetails(map[string]string{"combo_id": item.ComboID})
		}
		unit := combo.Price
		if hasMembership && combo.MemberPrice != nil {
			unit = *combo.MemberPrice
		}
		if branchID != "" {
			ok, err := s.combos.ReserveStock(tx, branchID, combo.ID, item.Quantity)
			if err != nil {
				return nil, 0, err
			}
			if !ok {
				return nil, 0, apperrors.Conflict("combo is out of stock at this branch").WithDetails(map[string]string{"combo_id": item.ComboID})
			}
		}
		total += unit * int64(item.Quantity)
		rows = append(rows, models.BookingCombo{ComboID: combo.ID, Quantity: item.Quantity, Price: unit})
	}
	return rows, total, nil
}

// releaseComboStock is the rollback pair to priceCombos' ReserveStock,
// called whenever a booking that reserved combo stock does not end up
// CONFIRMED (Phần 2.2, same pattern as voucher usage rollback).
func (s *bookingService) releaseComboStock(ctx context.Context, tx *gorm.DB, showtimeID, bookingID string) error {
	if s.combos == nil || s.branches == nil {
		return nil
	}
	showtime, err := s.repo.LockShowtime(ctx, tx, showtimeID)
	if err != nil || showtime == nil {
		return err
	}
	branchID, err := s.branches.HallBranch(tx, showtime.HallID)
	if err != nil {
		return err
	}
	if branchID == "" {
		return nil
	}
	return s.combos.ReleaseStockForBooking(ctx, tx, branchID, bookingID)
}

// voucherBase picks the amount a voucher's discount is computed against.
func voucherBase(scope string, seatTotal, comboTotal int64) int64 {
	switch scope {
	case models.VoucherScopeSeatOnly:
		return seatTotal
	case models.VoucherScopeComboOnly:
		return comboTotal
	default:
		return seatTotal + comboTotal
	}
}

// lockAndValidateVoucher locks the voucher row (part of the advisory -> user
// -> showtime -> seats -> voucher lock order) and validates it, returning a
// generic ErrVoucherInvalid on any failure — not found, expired, exhausted,
// per-user cap reached or below min order — deliberately indistinguishable,
// so a hold request can not be used to probe for valid codes
// (ADVANCED_FEATURES_DISCUSSION.md Phần 1.1 "chống dò mã").
func (s *bookingService) lockAndValidateVoucher(tx *gorm.DB, code, userID, hallID string, now time.Time, seatTotal, comboTotal int64) (*models.Voucher, int64, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	v, err := s.vouchers.LockByCode(tx, code)
	if err != nil {
		return nil, 0, err
	}
	if v == nil {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	if v.Status != models.VoucherActive || now.Before(v.StartsAt) || !now.Before(v.EndsAt) || v.UsageCount >= v.MaxUsage {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	// A loyalty milestone voucher is locked to the user who earned it.
	if v.AssignedUserID != nil && *v.AssignedUserID != userID {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	// A branch-scoped voucher only applies to a booking whose showtime's
	// hall is in that branch (Phần 1.1).
	if v.BranchID != nil && s.branches != nil {
		hallBranch, err := s.branches.HallBranch(tx, hallID)
		if err != nil {
			return nil, 0, err
		}
		if hallBranch != *v.BranchID {
			return nil, 0, apperrors.ErrVoucherInvalid
		}
	}
	userCount, err := s.vouchers.CountUserRedemptions(tx, v.ID, userID)
	if err != nil {
		return nil, 0, err
	}
	if userCount >= int64(v.MaxUsagePerUser) {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	base := voucherBase(v.ApplyScope, seatTotal, comboTotal)
	if base < v.MinOrderAmount || base <= 0 {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	discount := computeVoucherDiscount(v, base)
	if discount <= 0 {
		return nil, 0, apperrors.ErrVoucherInvalid
	}
	return v, discount, nil
}

func computeVoucherDiscount(v *models.Voucher, base int64) int64 {
	var discount int64
	if v.DiscountType == models.VoucherDiscountPercentage {
		discount = int64(math.Round(float64(base) * float64(v.DiscountValue) / 100))
	} else {
		discount = v.DiscountValue
	}
	if v.MaxDiscount != nil && discount > *v.MaxDiscount {
		discount = *v.MaxDiscount
	}
	if discount > base {
		discount = base
	}
	if discount < 0 {
		discount = 0
	}
	return discount
}

func holdable(seat models.ShowtimeSeat, userID string, own map[string]int64, now time.Time) bool {
	switch seat.Status {
	case models.SeatStatusAvailable:
		return true
	case models.SeatStatusHeld:
		if seat.HeldUntil != nil && !seat.HeldUntil.After(now) {
			return true
		}
		version, mine := own[seat.ID]
		return mine && seat.HeldBy != nil && *seat.HeldBy == userID && version == seat.Version
	default:
		return false
	}
}

func (s *bookingService) holdResponse(ctx context.Context, tx *gorm.DB, b *models.Booking) (*dto.HoldResponse, error) {
	bseats, err := s.repo.BookingSeats(ctx, tx, b.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(bseats))
	for _, bs := range bseats {
		ids = append(ids, bs.ShowtimeSeatID)
	}
	locked, err := s.repo.LockSeats(ctx, tx, b.ShowtimeID, ids)
	if err != nil {
		return nil, err
	}
	physByShowSeat := make(map[string]string, len(locked))
	physIDs := make([]string, 0, len(locked))
	for _, seat := range locked {
		physByShowSeat[seat.ID] = seat.SeatID
		physIDs = append(physIDs, seat.SeatID)
	}
	phys, err := s.repo.SeatsByIDs(ctx, tx, physIDs)
	if err != nil {
		return nil, err
	}
	held := make([]dto.HeldSeat, 0, len(bseats))
	for _, bs := range bseats {
		meta := phys[physByShowSeat[bs.ShowtimeSeatID]]
		held = append(held, dto.HeldSeat{
			ShowtimeSeatID: bs.ShowtimeSeatID,
			Label:          dto.SeatLabel(meta.RowLabel, meta.ColNumber),
			RowLabel:       meta.RowLabel,
			ColNumber:      meta.ColNumber,
			SeatType:       bs.SeatType,
			Price:          bs.Price,
		})
	}
	return &dto.HoldResponse{
		BookingID:   b.ID,
		ShowtimeID:  b.ShowtimeID,
		TotalAmount: b.TotalAmount,
		ExpiresAt:   *b.ExpiresAt,
		Seats:       held,
	}, nil
}

func (s *bookingService) Status(ctx context.Context, userID, bookingID string) (*dto.OrderStatusResponse, error) {
	b, err := s.ownedBooking(ctx, userID, bookingID)
	if err != nil {
		return nil, err
	}
	pay, err := s.paymentFor(ctx, b)
	if err != nil {
		return nil, err
	}
	show, err := s.showtimeOf(ctx, b.ShowtimeID)
	if err != nil {
		return nil, err
	}
	st := orderStatus(b, pay, show)
	return &st, nil
}

func (s *bookingService) Order(ctx context.Context, userID, bookingID string) (*dto.OrderDetailResponse, error) {
	b, err := s.ownedBooking(ctx, userID, bookingID)
	if err != nil {
		return nil, err
	}
	return s.orderDetail(ctx, b)
}

// AdminOrder is Order without the ownership check, for admin/staff customer
// support looking up any booking by id.
func (s *bookingService) AdminOrder(ctx context.Context, bookingID string) (*dto.OrderDetailResponse, error) {
	b, err := s.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, apperrors.ErrBookingNotFound
	}
	if err := s.reconcile(ctx, b); err != nil {
		return nil, err
	}
	b, err = s.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	return s.orderDetail(ctx, b)
}

// ownedBooking reconciles before returning; another user's booking is 403, not 404.
func (s *bookingService) ownedBooking(ctx context.Context, userID, bookingID string) (*models.Booking, error) {
	b, err := s.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, apperrors.ErrBookingNotFound
	}
	if b.UserID != userID {
		return nil, apperrors.Forbidden("this order belongs to another user")
	}
	if err := s.reconcile(ctx, b); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, bookingID)
}

func (s *bookingService) List(ctx context.Context, userID string, q dto.PageQuery) ([]dto.OrderStatusResponse, int64, error) {
	bookings, total, err := s.repo.ListByUser(ctx, userID, q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(bookings))
	for i := range bookings {
		ids = append(ids, bookings[i].ShowtimeID)
	}
	shows, err := s.repo.ShowtimeInfos(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.OrderStatusResponse, 0, len(bookings))
	for i := range bookings {
		var show *repository.ShowtimeInfoRow
		if info, ok := shows[bookings[i].ShowtimeID]; ok {
			show = &info
		}
		items = append(items, orderStatus(&bookings[i], nil, show))
	}
	return items, total, nil
}

func (s *bookingService) showtimeOf(ctx context.Context, showtimeID string) (*repository.ShowtimeInfoRow, error) {
	infos, err := s.repo.ShowtimeInfos(ctx, []string{showtimeID})
	if err != nil {
		return nil, err
	}
	if info, ok := infos[showtimeID]; ok {
		return &info, nil
	}
	return nil, nil
}

// Cancel leaves a checkout still open at a provider alone: if it gets paid later,
// the money arrives for an expired booking and is refunded.
func (s *bookingService) Cancel(ctx context.Context, userID, bookingID string) (*dto.OrderStatusResponse, error) {
	var (
		released []string
		showID   string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		released, showID = nil, ""
		b, err := s.repo.LockBooking(ctx, tx, bookingID)
		if err != nil {
			return err
		}
		if b == nil || b.UserID != userID {
			return apperrors.ErrBookingNotFound
		}
		showID = b.ShowtimeID
		if b.Status != models.BookingPending {
			return notPayable(b)
		}
		if b.PaidAt != nil {
			return apperrors.ErrPaymentInProgress
		}
		bseats, err := s.repo.BookingSeats(ctx, tx, b.ID)
		if err != nil {
			return err
		}
		for _, bs := range bseats {
			n, err := s.repo.ReleaseHeldSeat(ctx, tx, bs.ShowtimeSeatID, userID, bs.HoldVersion)
			if err != nil {
				return err
			}
			if n > 0 {
				released = append(released, bs.ShowtimeSeatID)
			}
		}
		n, err := s.repo.ExpireBooking(ctx, tx, b.ID, models.ReasonCanceled)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.Internal("booking changed under lock")
		}
		if s.vouchers != nil && b.VoucherID != nil {
			if err := s.vouchers.ReleaseUsage(ctx, tx, b.ID); err != nil {
				return err
			}
		}
		if err := s.releaseComboStock(ctx, tx, b.ShowtimeID, b.ID); err != nil {
			return err
		}
		return s.audit(ctx, tx, "orders.cancel", "booking", b.ID, b.ID,
			map[string]any{"status": models.BookingExpired, "reason": models.ReasonCanceled, "released_seats": len(released)})
	})
	if err != nil {
		return nil, err
	}
	s.broadcast(showID, models.SeatStatusAvailable, released)

	b, err := s.repo.FindByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	pay, err := s.paymentFor(ctx, b)
	if err != nil {
		return nil, err
	}
	show, err := s.showtimeOf(ctx, b.ShowtimeID)
	if err != nil {
		return nil, err
	}
	st := orderStatus(b, pay, show)
	return &st, nil
}

func (s *bookingService) paymentFor(ctx context.Context, b *models.Booking) (*models.Payment, error) {
	if b.PaymentID != nil {
		return s.payments.FindByID(ctx, *b.PaymentID)
	}
	return s.payments.LatestForBooking(ctx, b.ID)
}

const (
	defaultCheckinOpenBefore = 30 * time.Minute
	defaultCheckinCloseAfter = 20 * time.Minute
)

// Redeem takes a ticket id or QR code. Only an ISSUED ticket of this showtime inside its
// check-in window flips to REDEEMED, exactly once; a closed showtime still lets its tickets in.
// Any other verdict changes nothing and is audited as a failure.
func (s *bookingService) Redeem(ctx context.Context, ticketRef, showtimeID, staffUserID string) (*dto.RedeemResponse, error) {
	row, err := s.repo.TicketForGate(ctx, ticketRef)
	if err != nil {
		return nil, err
	}
	var staffBranchID string
	if staffUserID != "" {
		staffBranchID, err = s.repo.StaffBranch(ctx, staffUserID)
		if err != nil {
			return nil, err
		}
	}
	if row == nil || row.BookingStatus != models.BookingConfirmed {
		res := &dto.RedeemResponse{Status: models.RedeemNotFound}
		s.auditScan(ctx, "", "", showtimeID, res.Status)
		return res, nil
	}
	startAt := row.StartAt
	opensAt, closesAt := startAt.Add(-s.checkinOpenBefore), startAt.Add(s.checkinCloseAfter)
	res := &dto.RedeemResponse{
		TicketID:        row.ID,
		ShowtimeID:      row.ShowtimeID,
		MovieTitle:      row.MovieTitle,
		AgeRating:       row.AgeRating,
		HallName:        row.HallName,
		SeatLabel:       dto.SeatLabel(row.RowLabel, row.ColNumber),
		StartAt:         &startAt,
		CheckinOpensAt:  &opensAt,
		CheckinClosesAt: &closesAt,
	}
	now, err := s.repo.Now(ctx, nil)
	if err != nil {
		return nil, err
	}
	switch {
	case row.ShowtimeID != showtimeID:
		res.Status = models.RedeemWrongShow
	case staffBranchID != "" && row.BranchID != staffBranchID:
		res.Status = models.RedeemWrongBranch
	case row.Status != models.TicketIssued:
		res.Status = models.RedeemUsed
	case now.Before(opensAt):
		res.Status = models.RedeemTooEarly
	case now.After(closesAt):
		res.Status = models.RedeemClosed
	}
	if res.Status != "" {
		s.auditScan(ctx, row.ID, row.BookingID, showtimeID, res.Status)
		return res, nil
	}

	used := false
	err = s.inTx(ctx, func(tx *gorm.DB) error {
		n, err := s.repo.RedeemTicket(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		used = n == 0
		if used {
			return nil
		}
		return s.audit(ctx, tx, "staff.redeem_ticket", "ticket", row.ID, row.BookingID,
			map[string]any{"status": models.TicketRedeemed, "showtime_id": showtimeID})
	})
	if err != nil {
		return nil, err
	}
	res.Status = models.RedeemOK
	if used {
		res.Status = models.RedeemUsed
		s.auditScan(ctx, row.ID, row.BookingID, showtimeID, res.Status)
	}
	return res, nil
}

// auditScan never logs the scanned code: the ticket id, when known, stands for it.
func (s *bookingService) auditScan(ctx context.Context, ticketID, bookingID, showtimeID, verdict string) {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		rec = audit.Record{ActorRole: "system"}
	}
	rec.Action = "staff.redeem_ticket"
	rec.ResourceType = "ticket"
	rec.ResourceID = ticketID
	rec.BookingID = bookingID
	rec.Before = nil
	rec.After = map[string]any{"showtime_id": showtimeID, "verdict": verdict}
	rec.Outcome = audit.OutcomeFailure
	rec.ErrorMessage = verdict
	if err := audit.In(context.WithoutCancel(ctx), s.db, rec); err != nil {
		logger.Warn("refused ticket scan not audited", logger.Err(err))
	}
}

func (s *bookingService) sameSeats(ctx context.Context, tx *gorm.DB, bookingID string, seatIDs []string) (bool, error) {
	held, err := s.repo.BookingSeats(ctx, tx, bookingID)
	if err != nil {
		return false, err
	}
	if len(held) != len(seatIDs) {
		return false, nil
	}
	for _, bs := range held {
		if !slices.Contains(seatIDs, bs.ShowtimeSeatID) {
			return false, nil
		}
	}
	return true, nil
}

func (s *bookingService) broadcast(showtimeID, status string, ids []string) {
	if s.hub == nil || len(ids) == 0 {
		return
	}
	updates := make([]sse.SeatUpdate, 0, len(ids))
	for _, id := range ids {
		updates = append(updates, sse.SeatUpdate{ID: id, Status: status})
	}
	s.hub.Broadcast(showtimeID, sse.SeatEvent{ShowtimeID: showtimeID, Seats: updates})
}

// audit writes a success row for an order-lifecycle event. bookingID is the
// stable correlation key (see internal/audit doc comment) — pass it even
// when resourceType/resourceID is "payment" or "ticket", not just "booking".
func (s *bookingService) audit(ctx context.Context, tx *gorm.DB, action, resourceType, resourceID, bookingID string, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		rec = audit.Record{ActorRole: "system"}
	}
	rec.Action = action
	rec.ResourceType = resourceType
	rec.ResourceID = resourceID
	rec.BookingID = bookingID
	rec.Before = nil
	rec.After = after
	rec.Outcome = audit.OutcomeSuccess
	rec.ErrorMessage = ""
	return audit.In(ctx, tx, rec)
}

// auditRejected runs outside any transaction so forged callbacks still leave a trace.
func (s *bookingService) auditRejected(ctx context.Context, provider, txnRef, message string) {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		rec = audit.Record{ActorRole: "system"}
	}
	rec.Action = "payments.notify"
	rec.ResourceType = "payment"
	rec.ResourceID = txnRef
	rec.After = map[string]any{"provider": provider}
	rec.Outcome = audit.OutcomeFailure
	rec.ErrorMessage = message
	if err := audit.In(context.WithoutCancel(ctx), s.db, rec); err != nil {
		logger.Warn("rejected payment notification not audited", logger.Err(err))
	}
}

func (s *bookingService) bookingCombos(ctx context.Context, bookingID string) ([]dto.BookingComboResponse, error) {
	rows, err := s.combos.BookingCombos(ctx, nil, bookingID)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ComboID)
	}
	names, err := s.combos.FindByIDs(ctx, nil, ids)
	if err != nil {
		return nil, err
	}
	out := make([]dto.BookingComboResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.BookingComboResponse{
			ComboID: r.ComboID, Name: names[r.ComboID].Name, Quantity: r.Quantity,
			Price: r.Price, Delivered: r.Delivered, DeliveredAt: r.DeliveredAt,
		})
	}
	return out, nil
}

func (s *bookingService) orderDetail(ctx context.Context, b *models.Booking) (*dto.OrderDetailResponse, error) {
	rows, err := s.repo.TicketRows(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	pay, err := s.paymentFor(ctx, b)
	if err != nil {
		return nil, err
	}
	show, err := s.showtimeOf(ctx, b.ShowtimeID)
	if err != nil {
		return nil, err
	}
	res := &dto.OrderDetailResponse{
		OrderStatusResponse: orderStatus(b, pay, show),
		Tickets:             make([]dto.TicketResponse, 0, len(rows)),
	}
	if s.combos != nil {
		combos, err := s.bookingCombos(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		res.Combos = combos
	}
	for _, t := range rows {
		res.Tickets = append(res.Tickets, dto.TicketResponse{
			ID:             t.ID,
			ShowtimeSeatID: t.ShowtimeSeatID,
			SeatLabel:      dto.SeatLabel(t.RowLabel, t.ColNumber),
			SeatType:       t.SeatType,
			Price:          t.Price,
			Code:           t.Code,
			Status:         t.Status,
		})
	}
	return res, nil
}

// CounterSell sells tickets at the till to a walk-in: cash is collected, the
// booking is confirmed immediately and no ticket email goes out. Seats are sold
// straight from 'available' under row locks, so an online hold or finalize on
// the same seat can not race past it.
func (s *bookingService) CounterSell(ctx context.Context, req dto.CounterSellRequest) (*dto.OrderDetailResponse, error) {
	seatIDs := uniqueStrings(req.SeatIDs)
	if len(seatIDs) <= 0 || len(req.SeatIDs) != len(seatIDs) {
		return nil, apperrors.Validation("duplicate or missing seats in the request")
	}
	if len(seatIDs) > s.maxSeats {
		return nil, apperrors.ErrSeatLimitExceeded.WithDetails(map[string]string{"max": strconv.Itoa(s.maxSeats)})
	}

	var (
		booking models.Booking
		showID  string
		sold    []string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		now, err := s.repo.Now(ctx, tx)
		if err != nil {
			return err
		}
		showtime, err := s.repo.LockShowtime(ctx, tx, req.ShowID)
		if err != nil {
			return err
		}
		if showtime == nil {
			return apperrors.ErrShowtimeNotFound
		}
		if showtime.Status != models.ShowtimeOpen || !showtime.StartAt.After(now) {
			return apperrors.ErrShowtimeClosed
		}
		showing, err := s.repo.MovieShowing(ctx, tx, showtime.MovieID)
		if err != nil {
			return err
		}
		if !showing {
			return apperrors.ErrShowtimeClosed
		}

		locked, err := s.repo.LockSeats(ctx, tx, showtime.ID, seatIDs)
		if err != nil {
			return err
		}
		byID := make(map[string]models.ShowtimeSeat, len(locked))
		physIDs := make([]string, 0, len(seatIDs))
		for _, seat := range locked {
			byID[seat.ID] = seat
			physIDs = append(physIDs, seat.SeatID)
		}
		phys, err := s.repo.SeatsByIDs(ctx, tx, physIDs)
		if err != nil {
			return err
		}
		prices, err := s.repo.PricesByHall(ctx, tx, showtime.HallID)
		if err != nil {
			return err
		}

		var taken, gaps []string
		for _, id := range seatIDs {
			seat, ok := byID[id]
			if !ok {
				return apperrors.Validation("seat does not belong to this showtime").
					WithDetails(map[string]string{"seat_id": id})
			}
			label := dto.SeatLabel(phys[seat.SeatID].RowLabel, phys[seat.SeatID].ColNumber)
			switch {
			case phys[seat.SeatID].IsGap:
				gaps = append(gaps, label)
			case seat.Status != models.SeatStatusAvailable:
				taken = append(taken, label)
			}
		}
		if len(gaps) > 0 {
			return apperrors.ErrSeatNotSellable.WithDetails(map[string]string{"seats": strings.Join(gaps, ",")})
		}
		if len(taken) > 0 {
			return apperrors.ErrSeatTaken.WithDetails(map[string]string{"seats": strings.Join(taken, ",")})
		}

		var total int64
		bseats := make([]models.BookingSeat, 0, len(seatIDs))
		labels := make([]string, 0, len(seatIDs))
		for _, id := range seatIDs {
			seatType := phys[byID[id].SeatID].SeatType
			price, ok := prices[seatType]
			if !ok || price <= 0 {
				return apperrors.ErrMissingHallPrice.WithDetails(map[string]string{"seat_type": seatType})
			}
			total += price
			labels = append(labels, dto.SeatLabel(phys[byID[id].SeatID].RowLabel, phys[byID[id].SeatID].ColNumber))
			bseats = append(bseats, models.BookingSeat{
				ShowtimeSeatID: id,
				SeatType:       seatType,
				Price:          price,
			})
		}

		booking = models.Booking{
			ShowtimeID:    showtime.ID,
			CustomerName:  strings.TrimSpace(req.CustomerName),
			CustomerPhone: strings.TrimSpace(req.CustomerPhone),
			Status:        models.BookingPending,
			TotalAmount:   total,
		}
		if err := s.repo.CreateCounterBooking(ctx, tx, &booking); err != nil {
			return err
		}
		for i := range bseats {
			bseats[i].BookingID = booking.ID
		}
		if err := s.repo.CreateBookingSeats(ctx, tx, bseats); err != nil {
			return err
		}

		for _, bs := range bseats {
			n, err := s.repo.SellSeatAtCounter(ctx, tx, bs.ShowtimeSeatID)
			if err != nil {
				return err
			}
			if n == 0 {
				return fmt.Errorf("counter sell %s: seat changed under row lock", bs.ShowtimeSeatID)
			}
			sold = append(sold, bs.ShowtimeSeatID)
		}

		tickets := make([]models.Ticket, 0, len(bseats))
		for _, bs := range bseats {
			tickets = append(tickets, models.Ticket{
				BookingID:      booking.ID,
				ShowtimeSeatID: bs.ShowtimeSeatID,
				Price:          bs.Price,
				Code:           newTicketCode(),
				Status:         models.TicketIssued,
			})
		}
		if err := s.repo.CreateTickets(ctx, tx, tickets); err != nil {
			return err
		}
		n, err := s.repo.ConfirmBooking(ctx, tx, booking.ID)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("confirm counter booking %s: row changed under lock", booking.ID)
		}
		booking.Status = models.BookingConfirmed
		showID = showtime.ID
		if s.ledger != nil {
			// Cash collected at the till: no payment attempt row, so the
			// ledger references the booking itself.
			if err := s.ledger.Write(tx, &models.LedgerEntry{
				Type: models.LedgerPaymentCaptured, Amount: total,
				ReferenceTable: "bookings", ReferenceID: booking.ID, UserID: nullableUserID(booking.UserID), BookingID: &booking.ID,
			}); err != nil {
				return err
			}
		}
		if s.loyalty != nil {
			// booking.UserID is empty unless a counter sale is attached to a
			// known account — EarnForBooking already no-ops on an empty id.
			if err := s.loyalty.EarnForBooking(ctx, tx, booking.UserID, total, len(tickets), booking.ID); err != nil {
				return err
			}
		}
		return s.audit(ctx, tx, "orders.counter_sell", "booking", booking.ID, booking.ID, map[string]any{
			"status": models.BookingConfirmed, "tickets": len(tickets), "total": total, "seats": labels,
		})
	})
	if err != nil {
		return nil, err
	}
	s.broadcast(showID, "sold", sold)
	b, err := s.repo.FindByID(context.WithoutCancel(ctx), booking.ID)
	if err != nil {
		return nil, err
	}
	return s.orderDetail(ctx, b)
}

func orderStatus(b *models.Booking, pay *models.Payment, show *repository.ShowtimeInfoRow) dto.OrderStatusResponse {
	res := dto.OrderStatusResponse{
		ID:           b.ID,
		ShowtimeID:   b.ShowtimeID,
		Status:       b.Status,
		StatusReason: derefString(b.StatusReason),
		TotalAmount:  b.TotalAmount,
		CreatedAt:    b.CreatedAt,
		ExpiresAt:    b.ExpiresAt,
		PaidAt:       b.PaidAt,
	}
	if show != nil {
		now := time.Now()
		res.Showtime = &dto.OrderShowtime{
			MovieID:    show.MovieID,
			MovieTitle: show.MovieTitle,
			AgeRating:  show.AgeRating,
			HallID:     show.HallID,
			HallName:   show.HallName,
			StartAt:    show.StartAt,
			EndAt:      show.EndAt,
			Started:    !show.StartAt.After(now),
			Ended:      show.EndAt.Before(now),
		}
	}
	if pay != nil {
		res.Payment = &dto.PaymentSummary{
			ID:           pay.ID,
			Provider:     pay.Provider,
			TxnRef:       pay.TxnRef,
			Status:       pay.Status,
			StatusReason: derefString(pay.StatusReason),
			Amount:       pay.Amount,
			PaidAmount:   pay.PaidAmount,
			PaidAt:       pay.PaidAt,
			RefundedAt:   pay.RefundedAt,
		}
	}
	return res
}

// nullableUserID: a counter sale has no account (UserID empty); the ledger
// column is nullable for exactly that case.
func nullableUserID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("read rand: " + err.Error())
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}

func newTicketCode() string { return randomHex(16) }
