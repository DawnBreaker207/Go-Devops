package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

const (
	// How long past the hold an unpaid attempt is still queried before it is abandoned.
	abandonAfter = 15 * time.Minute
	// How long a stored attempt may wait for its checkout URL before a new pay request takes it over.
	orphanCheckoutAfter = 30 * time.Second
	providerCallTimeout = 20 * time.Second
	// refundLease outlives refundTimeout so two workers never call the provider at once.
	refundLease   = 2 * time.Minute
	refundTimeout = 30 * time.Second
	// Failed tries after which a refund or paid booking raises an alert; retries go on.
	stuckAlertAttempts       = 8
	publishTimeout           = 3 * time.Second
	defaultLateCaptureWindow = 24 * time.Hour
)

// Pay stores the attempt row before calling the provider (so an immediate IPN finds it); never calls inside a transaction.
func (s *bookingService) Pay(ctx context.Context, userID, bookingID string, req dto.PayRequest) (*dto.PayResponse, error) {
	name := strings.TrimSpace(req.Provider)
	if name == "" {
		name = s.providers.Default()
	}
	if name == "" {
		return nil, apperrors.Validation("provider is required")
	}
	provider, ok := s.providers.Get(name)
	if !ok {
		return nil, apperrors.Validation("payment provider is not available").WithDetails(map[string]string{"provider": name})
	}

	var (
		attempt *models.Payment
		reused  bool
		resume  bool
		booking models.Booking
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		attempt, reused, resume = nil, false, false
		// Account row before booking row (same order as hold) so locks can't deadlock.
		active, err := s.repo.UserActive(ctx, tx, userID)
		if err != nil {
			return err
		}
		b, err := s.repo.LockBooking(ctx, tx, bookingID)
		if err != nil {
			return err
		}
		if b == nil || b.UserID != userID {
			return apperrors.ErrBookingNotFound
		}
		if !active {
			return apperrors.ErrAccountLocked
		}
		if b.Status != models.BookingPending {
			return notPayable(b)
		}
		if b.PaidAt != nil {
			return apperrors.ErrBookingNotPending
		}
		now, err := s.repo.Now(ctx, tx)
		if err != nil {
			return err
		}
		if b.ExpiresAt == nil || !b.ExpiresAt.After(now) {
			return apperrors.ErrBookingExpired
		}
		// No money for a stopped/started show: confirm could only refund it.
		showtime, err := s.repo.LockShowtime(ctx, tx, b.ShowtimeID)
		if err != nil {
			return err
		}
		if showtime == nil || showtime.Status != models.ShowtimeOpen || !showtime.StartAt.After(now) {
			return apperrors.ErrShowtimeClosed
		}
		// Zero total means no seats (prices are always positive): an init shell can't start payment.
		if b.TotalAmount <= 0 {
			return apperrors.ErrBookingEmpty
		}
		booking = *b

		open, err := s.payments.LockOpen(ctx, tx, b.ID, name)
		if err != nil {
			return err
		}
		if open != nil {
			attempt, reused = open, true
			if open.RedirectURL == nil {
				// Stored but never opened (call failed/died): after a grace period this request takes it over.
				n, err := s.payments.ClaimOrphanCheckout(ctx, tx, open.ID, orphanCheckoutAfter)
				if err != nil {
					return err
				}
				resume = n > 0
			}
			return nil
		}
		attempt = &models.Payment{
			BookingID: b.ID,
			Provider:  name,
			TxnRef:    newTxnRef(),
			// Payable(), NOT TotalAmount: a discount must reach the gateway, and
			// total_amount deliberately stays the undiscounted seat subtotal so
			// finalizeTx's seat-price assertion still holds. Amount-mismatch
			// detection and refunds both read this column, so they follow.
			Amount:    b.Payable(),
			Status:    models.PaymentPending,
			ExpiresAt: b.ExpiresAt,
		}
		if err := s.payments.Create(ctx, tx, attempt); err != nil {
			return err
		}
		return s.audit(ctx, tx, "orders.pay", "payment", attempt.ID, b.ID, map[string]any{
			"booking_id": b.ID, "provider": name, "txn_ref": attempt.TxnRef, "amount": attempt.Amount,
		})
	})
	if err != nil {
		return nil, err
	}
	if reused && !resume {
		if attempt.RedirectURL == nil {
			return nil, apperrors.Conflict("payment is still being created, retry shortly")
		}
		return payResponse(attempt, *attempt.RedirectURL), nil
	}

	createCtx, cancel := context.WithTimeout(ctx, providerCallTimeout)
	checkout, err := provider.CreatePayment(createCtx, payment.CreateRequest{
		TxnRef:      attempt.TxnRef,
		BookingID:   booking.ID,
		Amount:      attempt.Amount,
		Description: "Thanh toan ve xem phim " + booking.ID[:8],
		ClientIP:    req.ClientIP,
		Locale:      "vn",
		ReturnURL:   s.callbackURL(name, "return"),
		NotifyURL:   s.callbackURL(name, "ipn"),
		ExpiresAt:   *booking.ExpiresAt,
	})
	cancel()
	bg := context.WithoutCancel(ctx)
	if err != nil && resume {
		// Retrying an orphan keeps the attempt open; the sweep abandons it if the provider never answers.
		return nil, apperrors.ErrPaymentGateway.Wrap(err)
	}
	if err != nil {
		createErr := err
		if ferr := s.inTx(bg, func(tx *gorm.DB) error {
			if _, err := s.payments.MarkFailed(bg, tx, attempt.ID, models.PaymentReasonCreateFailed); err != nil {
				return err
			}
			return s.audit(bg, tx, "payments.create_failed", "payment", attempt.ID, booking.ID,
				map[string]any{"booking_id": booking.ID, "provider": name, "error": createErr.Error()})
		}); ferr != nil {
			logger.Warn("failed checkout not recorded", logger.String("payment_id", attempt.ID), logger.Err(ferr))
		}
		return nil, apperrors.ErrPaymentGateway.Wrap(createErr) // booking untouched: pay again later
	}
	if err := s.payments.SetCheckout(bg, attempt.ID, checkout.RedirectURL, checkout.ProviderTxnID, checkout.Data); err != nil {
		return nil, err
	}
	return payResponse(attempt, checkout.RedirectURL), nil
}

func payResponse(p *models.Payment, redirectURL string) *dto.PayResponse {
	return &dto.PayResponse{
		PaymentID:   p.ID,
		Provider:    p.Provider,
		TxnRef:      p.TxnRef,
		RedirectURL: redirectURL,
		ExpiresAt:   p.ExpiresAt,
	}
}

func (s *bookingService) callbackURL(provider, kind string) string {
	return fmt.Sprintf("%s/api/v1/payments/%s/%s", s.publicBaseURL, url.PathEscape(provider), kind)
}

func notPayable(b *models.Booking) error {
	switch b.Status {
	case models.BookingExpired:
		return apperrors.ErrBookingExpired
	case models.BookingRefunded:
		return apperrors.ErrBookingRefunded
	default:
		return apperrors.ErrBookingNotPending
	}
}

// HandleNotification: duplicates are no-ops; a database failure asks the provider to retry.
func (s *bookingService) HandleNotification(ctx context.Context, provider payment.Provider, r *http.Request) payment.AckStatus {
	n, err := provider.ParseNotification(r)
	switch {
	case errors.Is(err, payment.ErrInvalidSignature):
		s.auditRejected(ctx, provider.Name(), "", "invalid signature")
		return payment.AckInvalidSignature
	case err != nil:
		s.auditRejected(ctx, provider.Name(), "", err.Error())
		return payment.AckInvalid
	}
	ack, err := s.applyNotification(ctx, provider.Name(), n, "webhook:"+provider.Name())
	if err != nil {
		logger.Error("payment notification not applied; provider will retry",
			logger.String("provider", provider.Name()), logger.String("txn_ref", n.TxnRef), logger.Err(err))
		return payment.AckRetryLater
	}
	if ack == payment.AckUnknownTxn {
		s.auditRejected(ctx, provider.Name(), n.TxnRef, "unknown transaction")
	}
	return ack
}

// HandleReturn trusts the redirect only to identify the attempt; state is queried from the provider.
func (s *bookingService) HandleReturn(ctx context.Context, provider payment.Provider, r *http.Request) (*dto.PaymentReturnResponse, error) {
	ref, err := provider.ParseReturn(r)
	if errors.Is(err, payment.ErrInvalidSignature) {
		return nil, apperrors.ErrInvalidSignature
	}
	if err != nil {
		return nil, apperrors.Validation("invalid payment return parameters")
	}
	attempt, err := s.payments.FindByRef(ctx, provider.Name(), ref)
	if err != nil {
		return nil, err
	}
	if attempt == nil {
		return nil, apperrors.NotFound("payment not found")
	}
	if _, err := s.reconcilePayment(ctx, attempt); err != nil {
		return nil, err
	}
	if attempt, err = s.payments.FindByID(ctx, attempt.ID); err != nil {
		return nil, err
	}
	b, err := s.repo.FindByID(ctx, attempt.BookingID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, apperrors.ErrBookingNotFound
	}
	return &dto.PaymentReturnResponse{
		BookingID:     b.ID,
		BookingStatus: b.Status,
		BookingReason: derefString(b.StatusReason),
		PaymentID:     attempt.ID,
		PaymentStatus: attempt.Status,
	}, nil
}

func (s *bookingService) applyNotification(ctx context.Context, providerName string, n *payment.Notification, source string) (payment.AckStatus, error) {
	var (
		ack       payment.AckStatus
		out       finalizeOutcome
		bookingID string
		showID    string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		ack, out, bookingID, showID = payment.AckProcessed, finalizeOutcome{}, "", ""

		// Booking row before payment row (same order as Pay) so pay and notification can't deadlock.
		found, err := s.payments.FindByRef(ctx, providerName, n.TxnRef)
		if err != nil {
			return err
		}
		if found == nil {
			ack = payment.AckUnknownTxn
			return nil
		}
		b, err := s.repo.LockBooking(ctx, tx, found.BookingID)
		if err != nil {
			return err
		}
		if b == nil {
			return fmt.Errorf("payment %s: booking %s missing", found.ID, found.BookingID)
		}
		attempt, err := s.payments.LockByRef(ctx, tx, providerName, n.TxnRef)
		if err != nil {
			return err
		}
		if attempt == nil || attempt.BookingID != b.ID {
			return fmt.Errorf("payment %s changed while locking its booking", found.ID)
		}
		bookingID, showID = b.ID, b.ShowtimeID
		switch attempt.Status {
		case models.PaymentPaid, models.PaymentRefundPending, models.PaymentRefunded:
			ack = payment.AckDuplicate
			return nil
		}

		switch n.Status {
		case payment.StatePending:
			return nil
		case payment.StateFailed:
			if attempt.Status != models.PaymentPending {
				return nil
			}
			if _, err := s.payments.MarkFailed(ctx, tx, attempt.ID, models.PaymentReasonDeclined); err != nil {
				return err
			}
			return s.audit(ctx, tx, "payments.failed", "payment", attempt.ID, bookingID,
				map[string]any{"booking_id": attempt.BookingID, "provider": providerName, "source": source})
		case payment.StatePaid:
		default:
			ack = payment.AckInvalid
			return nil
		}

		// Booking already carries another attempt's money (paid twice): this payment goes straight back.
		if b.PaidAt != nil {
			if _, err := s.payments.MarkCaptured(ctx, tx, attempt.ID, models.PaymentRefundPending,
				n.Amount, n.ProviderTxnID, n.Data, models.PaymentReasonDuplicate); err != nil {
				return err
			}
			out.refunds = append(out.refunds, attempt.ID)
			return s.audit(ctx, tx, "orders.refund", "payment", attempt.ID, bookingID, map[string]any{
				"booking_id": b.ID, "reason": models.PaymentReasonDuplicate, "amount": n.Amount, "source": source,
			})
		}

		if _, err := s.payments.MarkCaptured(ctx, tx, attempt.ID, models.PaymentPaid,
			n.Amount, n.ProviderTxnID, n.Data, ""); err != nil {
			return err
		}
		rows, err := s.repo.SetPaid(ctx, tx, b.ID, attempt.ID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("booking %s: paid state changed under lock", b.ID)
		}
		paymentID := attempt.ID
		b.PaymentID = &paymentID

		reason := ""
		if n.Amount != attempt.Amount {
			reason = models.ReasonAmountMismatch
		}
		return s.finalizeTx(ctx, tx, b, reason, source, &out)
	})
	if err != nil {
		return 0, err
	}
	s.afterFinalize(ctx, bookingID, showID, out)
	return ack, nil
}

// Confirm reconciles with the provider first, so a lost IPN does not block the ticket.
func (s *bookingService) Confirm(ctx context.Context, userID, bookingID string) (*dto.OrderDetailResponse, error) {
	b, err := s.ownedBooking(ctx, userID, bookingID)
	if err != nil {
		return nil, err
	}
	if b.TotalAmount <= 0 {
		return nil, apperrors.ErrBookingEmpty
	}
	settled, err := s.finalize(ctx, b.ID, sourceCustomer)
	if err != nil {
		return nil, err
	}
	switch settled.Status {
	case models.BookingConfirmed:
		return s.orderDetail(ctx, settled)
	case models.BookingRefunded:
		return nil, apperrors.ErrBookingRefunded.WithDetails(map[string]string{"reason": derefString(settled.StatusReason)})
	case models.BookingExpired:
		return nil, apperrors.ErrBookingExpired
	default:
		return nil, apperrors.ErrBookingNotPaid
	}
}

type finalizeOutcome struct {
	confirmed bool
	sold      []string
	released  []string
	refunds   []string // payment ids to refund at the provider after commit
}

func (s *bookingService) finalize(ctx context.Context, bookingID, source string) (*models.Booking, error) {
	var (
		out    finalizeOutcome
		showID string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		out, showID = finalizeOutcome{}, ""
		b, err := s.repo.LockBooking(ctx, tx, bookingID)
		if err != nil {
			return err
		}
		if b == nil {
			return apperrors.ErrBookingNotFound
		}
		showID = b.ShowtimeID
		if b.Status == models.BookingConfirmed || b.Status == models.BookingRefunded {
			return nil
		}
		if b.PaidAt == nil || b.PaymentID == nil {
			if b.Status == models.BookingExpired {
				return nil
			}
			return apperrors.ErrBookingNotPaid
		}
		return s.finalizeTx(ctx, tx, b, "", source, &out)
	})
	if err != nil {
		return nil, err
	}
	s.afterFinalize(ctx, bookingID, showID, out)
	return s.repo.FindByID(context.WithoutCancel(ctx), bookingID)
}

// finalizeTx: the only place a paid booking leaves PENDING. Sells held seats (fenced by hold
// version/TTL/showtime) or refunds in one transaction; provider refund after commit. The locked
// row makes concurrent IPN/reconcile/confirm converge; non-empty reason forces the refund.
func (s *bookingService) finalizeTx(ctx context.Context, tx *gorm.DB, b *models.Booking, reason, source string, out *finalizeOutcome) error {
	switch b.Status {
	case models.BookingConfirmed, models.BookingRefunded:
		return nil
	case models.BookingExpired:
		if reason == "" {
			reason = models.ReasonPaidAfterExpiry
		}
	}

	now, err := s.repo.Now(ctx, tx)
	if err != nil {
		return err
	}
	bseats, err := s.repo.BookingSeats(ctx, tx, b.ID)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(bseats))
	for _, bs := range bseats {
		ids = append(ids, bs.ShowtimeSeatID)
	}

	if reason == "" {
		showtime, err := s.repo.LockShowtime(ctx, tx, b.ShowtimeID)
		if err != nil {
			return err
		}
		switch {
		case showtime == nil || showtime.Status != models.ShowtimeOpen || !showtime.StartAt.After(now):
			reason = models.ReasonShowtimeClosed
		case b.ExpiresAt == nil || !b.ExpiresAt.After(now):
			reason = models.ReasonHoldExpired
		}
	}

	locked, err := s.repo.LockSeats(ctx, tx, b.ShowtimeID, ids)
	if err != nil {
		return err
	}
	if reason == "" {
		byID := make(map[string]models.ShowtimeSeat, len(locked))
		for _, seat := range locked {
			byID[seat.ID] = seat
		}
		if len(bseats) == 0 {
			reason = models.ReasonSeatsLost
		}
		for _, bs := range bseats {
			seat, ok := byID[bs.ShowtimeSeatID]
			if !ok || seat.Status != models.SeatStatusHeld || seat.HeldBy == nil || *seat.HeldBy != b.UserID ||
				seat.Version != bs.HoldVersion || seat.HeldUntil == nil || !seat.HeldUntil.After(now) {
				reason = models.ReasonSeatsLost // swept or taken over
				break
			}
		}
	}

	if reason != "" {
		return s.refundTx(ctx, tx, b, bseats, reason, source, out)
	}

	var total int64
	tickets := make([]models.Ticket, 0, len(bseats))
	for _, bs := range bseats {
		n, err := s.repo.SellSeat(ctx, tx, bs.ShowtimeSeatID, b.UserID, bs.HoldVersion)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("sell seat %s: fence moved under row lock", bs.ShowtimeSeatID)
		}
		total += bs.Price
		tickets = append(tickets, models.Ticket{
			BookingID:      b.ID,
			ShowtimeSeatID: bs.ShowtimeSeatID,
			Price:          bs.Price,
			Code:           newTicketCode(),
			Status:         models.TicketIssued,
		})
		out.sold = append(out.sold, bs.ShowtimeSeatID)
	}
	if total != b.TotalAmount {
		return fmt.Errorf("booking %s: seat prices %d do not add up to total %d", b.ID, total, b.TotalAmount)
	}
	if err := s.repo.CreateTickets(ctx, tx, tickets); err != nil {
		return err
	}
	n, err := s.repo.ConfirmBooking(ctx, tx, b.ID)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("confirm booking %s: row changed under lock", b.ID)
	}
	out.confirmed = true
	return s.audit(ctx, tx, "orders.confirm", "booking", b.ID, b.ID, map[string]any{
		"status": models.BookingConfirmed, "tickets": len(tickets), "total": total,
		"payment_id": derefString(b.PaymentID), "source": source,
	})
}

func (s *bookingService) refundTx(ctx context.Context, tx *gorm.DB, b *models.Booking, bseats []models.BookingSeat, reason, source string, out *finalizeOutcome) error {
	if b.PaymentID == nil {
		return fmt.Errorf("refund booking %s: no payment carries its money", b.ID)
	}
	for _, bs := range bseats {
		n, err := s.repo.ReleaseHeldSeat(ctx, tx, bs.ShowtimeSeatID, b.UserID, bs.HoldVersion)
		if err != nil {
			return err
		}
		if n > 0 {
			out.released = append(out.released, bs.ShowtimeSeatID)
		}
	}
	n, err := s.repo.RefundBooking(ctx, tx, b.ID, reason)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("refund booking %s: row changed under lock", b.ID)
	}
	if n, err = s.payments.MarkRefundPending(ctx, tx, *b.PaymentID, reason); err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("refund payment %s: not in paid state", *b.PaymentID)
	}
	out.refunds = append(out.refunds, *b.PaymentID)
	// resource_type="payment" matches the duplicate-capture site in applyNotification: both refund an
	// attempt, keyed off the payment with booking_id as correlation.
	return s.audit(ctx, tx, "orders.refund", "payment", *b.PaymentID, b.ID, map[string]any{
		// Payable(): the audited amount must be what was actually collected, not
		// the pre-discount subtotal.
		"status": models.BookingRefunded, "reason": reason, "amount": b.Payable(),
		"payment_id": *b.PaymentID, "source": source,
	})
}

// afterFinalize runs post-commit side effects; they must not die with the request context.
func (s *bookingService) afterFinalize(ctx context.Context, bookingID, showID string, out finalizeOutcome) {
	bg := context.WithoutCancel(ctx)
	for _, id := range out.refunds {
		s.settleRefund(bg, id)
	}
	s.broadcast(showID, models.SeatStatusAvailable, out.released)
	if out.confirmed {
		s.publishTicketEmail(bg, bookingID)
		s.broadcast(showID, models.SeatStatusSold, out.sold)
	}
}

// settleRefund: a failure leaves the refund pending; the sweep retries.
func (s *bookingService) settleRefund(ctx context.Context, paymentID string) bool {
	// Claim first: IPN path and sweep may both get here; provider must never be asked twice at once.
	attempt, err := s.payments.ClaimRefund(ctx, paymentID, refundLease)
	if err != nil {
		logger.Warn("claim refund failed", logger.String("payment_id", paymentID), logger.Err(err))
		return false
	}
	if attempt == nil {
		return false // settled, claimed by another worker, or waiting for its retry time
	}
	provider, ok := s.providers.Get(attempt.Provider)
	if !ok {
		s.deferRefund(ctx, attempt, "payment provider not enabled")
		return false
	}
	amount := attempt.Amount
	if attempt.PaidAmount != nil {
		amount = *attempt.PaidAmount
	}
	refundCtx, cancel := context.WithTimeout(ctx, refundTimeout)
	err = provider.Refund(refundCtx, transactionOf(attempt), payment.RefundRequest{Amount: amount, Reason: derefString(attempt.StatusReason)})
	cancel()
	if err != nil {
		s.deferRefund(ctx, attempt, err.Error())
		return false
	}
	err = s.inTx(ctx, func(tx *gorm.DB) error {
		n, err := s.payments.MarkRefunded(ctx, tx, attempt.ID)
		if err != nil || n == 0 {
			return err
		}
		return s.audit(ctx, tx, "payments.refunded", "payment", attempt.ID, attempt.BookingID, map[string]any{
			"booking_id": attempt.BookingID, "provider": attempt.Provider, "amount": amount,
			"reason": derefString(attempt.StatusReason),
		})
	})
	if err != nil {
		logger.Warn("refund done but not recorded", logger.String("payment_id", attempt.ID), logger.Err(err))
		return false
	}
	return true
}

func (s *bookingService) deferRefund(ctx context.Context, attempt *models.Payment, reason string) {
	logger.Warn("provider refund failed; will retry", logger.String("payment_id", attempt.ID),
		logger.String("attempt", fmt.Sprint(attempt.RefundAttempts)), logger.String("error", reason))
	if err := s.payments.DeferRefund(ctx, attempt.ID, reason); err != nil {
		logger.Warn("schedule refund retry failed", logger.String("payment_id", attempt.ID), logger.Err(err))
	}
	if attempt.RefundAttempts != stuckAlertAttempts {
		return
	}
	logger.Error("refund keeps failing, needs a look",
		logger.String("payment_id", attempt.ID), logger.String("provider", attempt.Provider))
	if err := s.audit(ctx, s.db, "payments.refund_stuck", "payment", attempt.ID, attempt.BookingID, map[string]any{
		"booking_id": attempt.BookingID, "provider": attempt.Provider,
		"attempts": attempt.RefundAttempts, "error": reason,
	}); err != nil {
		logger.Warn("audit stuck refund failed", logger.String("payment_id", attempt.ID), logger.Err(err))
	}
}

// reconcile queries providers (IPN may be lost) and settles paid bookings whose confirm never completed.
func (s *bookingService) reconcile(ctx context.Context, b *models.Booking) error {
	if b.Status == models.BookingPending && b.PaidAt != nil {
		_, err := s.finalize(ctx, b.ID, sourceReconcile)
		return err
	}
	if b.Status != models.BookingPending && b.Status != models.BookingExpired {
		return nil
	}
	attempts, err := s.payments.ReconcilableForBooking(ctx, b.ID, s.lateCaptureWindow)
	if err != nil {
		return err
	}
	for i := range attempts {
		if _, err := s.reconcilePayment(ctx, &attempts[i]); err != nil {
			return err
		}
	}
	return nil
}

type reconcileResult int

const (
	reconcileNothing reconcileResult = iota
	reconcileApplied
	reconcileAbandoned
)

func (s *bookingService) reconcilePayment(ctx context.Context, attempt *models.Payment) (reconcileResult, error) {
	switch attempt.Status {
	case models.PaymentPending, models.PaymentFailed:
	default:
		return reconcileNothing, nil
	}
	provider, ok := s.providers.Get(attempt.Provider)
	if !ok {
		return reconcileNothing, nil
	}
	n, qerr := provider.QueryStatus(ctx, transactionOf(attempt))
	if err := s.payments.TouchChecked(ctx, attempt.ID); err != nil {
		return reconcileNothing, err
	}
	if qerr != nil && !errors.Is(qerr, payment.ErrUnknownTxn) {
		logger.Warn("payment status query failed", logger.String("payment_id", attempt.ID), logger.Err(qerr))
		return reconcileNothing, nil
	}
	if attempt.Status == models.PaymentFailed {
		// Money collected after give-up: settling confirms or refunds. Never abandoned again.
		if qerr == nil && n.Status == payment.StatePaid {
			n.TxnRef = attempt.TxnRef
			if _, err := s.applyNotification(ctx, attempt.Provider, n, sourceReconcile); err != nil {
				return reconcileNothing, err
			}
			return reconcileApplied, nil
		}
		return reconcileNothing, nil
	}
	if qerr == nil && (n.Status == payment.StatePaid || n.Status == payment.StateFailed) {
		n.TxnRef = attempt.TxnRef
		if _, err := s.applyNotification(ctx, attempt.Provider, n, sourceReconcile); err != nil {
			return reconcileNothing, err
		}
		return reconcileApplied, nil
	}

	now, err := s.repo.Now(ctx, nil)
	if err != nil {
		return reconcileNothing, err
	}
	if attempt.ExpiresAt != nil && attempt.ExpiresAt.Add(abandonAfter).Before(now) {
		abandoned := false
		err := s.inTx(ctx, func(tx *gorm.DB) error {
			n, err := s.payments.MarkFailed(ctx, tx, attempt.ID, models.PaymentReasonAbandoned)
			if err != nil || n == 0 {
				return err
			}
			abandoned = true
			return s.audit(ctx, tx, "payments.abandoned", "payment", attempt.ID, attempt.BookingID,
				map[string]any{"booking_id": attempt.BookingID, "provider": attempt.Provider})
		})
		if err != nil {
			return reconcileNothing, err
		}
		if abandoned {
			return reconcileAbandoned, nil
		}
		return reconcileNothing, nil
	}
	return reconcileNothing, nil
}

// CancelShowtime: same MarkRefundPending-in-tx + post-commit settleRefund pipeline as every other
// refund (see finalizeTx/refundTx). CONFIRMED flips straight to REFUNDED with tickets voided
// (RefundBooking can't undo a sale); PENDING is released/expired, refunded too if already paid.
func (s *bookingService) CancelShowtime(ctx context.Context, showtimeID string) (*dto.ShowtimeCancelResponse, error) {
	var (
		refundPaymentIDs []string
		released         []string
		affected         int
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		refundPaymentIDs, released, affected = nil, nil, 0

		showtime, err := s.repo.LockShowtimeExclusive(ctx, tx, showtimeID)
		if err != nil {
			return err
		}
		if showtime == nil {
			return apperrors.ErrShowtimeNotFound
		}
		if showtime.Status == models.ShowtimeCancelled {
			return apperrors.ErrShowtimeAlreadyCancelled
		}

		ids, err := s.repo.BookingIDsForShowtime(ctx, tx, showtimeID, []string{models.BookingPending, models.BookingConfirmed})
		if err != nil {
			return err
		}
		for _, id := range ids {
			b, err := s.repo.LockBooking(ctx, tx, id)
			if err != nil {
				return err
			}
			if b == nil {
				continue
			}
			switch b.Status {
			case models.BookingPending:
				bseats, err := s.repo.BookingSeats(ctx, tx, b.ID)
				if err != nil {
					return err
				}
				for _, bs := range bseats {
					n, err := s.repo.ReleaseHeldSeat(ctx, tx, bs.ShowtimeSeatID, b.UserID, bs.HoldVersion)
					if err != nil {
						return err
					}
					if n > 0 {
						released = append(released, bs.ShowtimeSeatID)
					}
				}
				if b.PaidAt != nil && b.PaymentID != nil {
					// Rare: paid but never finalized. Refund the money and settle the
					// booking through the normal refund path.
					n, err := s.payments.MarkRefundPending(ctx, tx, *b.PaymentID, models.ReasonShowtimeCancelled)
					if err != nil {
						return err
					}
					if n > 0 {
						refundPaymentIDs = append(refundPaymentIDs, *b.PaymentID)
					}
					if _, err := s.repo.RefundBooking(ctx, tx, b.ID, models.ReasonShowtimeCancelled); err != nil {
						return err
					}
				} else {
					if _, err := s.repo.ExpireBooking(ctx, tx, b.ID, models.ReasonShowtimeCancelled); err != nil {
						return err
					}
				}
			case models.BookingConfirmed:
				if _, err := s.repo.VoidTicketsForBooking(ctx, tx, b.ID); err != nil {
					return err
				}
				if b.PaymentID != nil {
					n, err := s.payments.MarkRefundPending(ctx, tx, *b.PaymentID, models.ReasonShowtimeCancelled)
					if err != nil {
						return err
					}
					if n > 0 {
						refundPaymentIDs = append(refundPaymentIDs, *b.PaymentID)
					}
				}
				if _, err := s.repo.RefundConfirmedBooking(ctx, tx, b.ID, models.ReasonShowtimeCancelled); err != nil {
					return err
				}
			default:
				continue
			}
			affected++
			if err := s.audit(ctx, tx, "orders.refund", "booking", b.ID, b.ID, map[string]any{
				"status": models.BookingRefunded, "reason": models.ReasonShowtimeCancelled, "source": "admin_cancel_showtime",
			}); err != nil {
				return err
			}
		}

		if _, err := s.repo.CancelShowtimeStatus(ctx, tx, showtimeID); err != nil {
			return err
		}
		return s.audit(ctx, tx, "admin.cancel_showtime", "showtime", showtimeID, "", map[string]any{
			"status": models.ShowtimeCancelled, "bookings_affected": affected,
		})
	})
	if err != nil {
		return nil, err
	}
	bumpCatalog(ctx, s.catalogCache)

	bg := context.WithoutCancel(ctx)
	for _, id := range refundPaymentIDs {
		s.settleRefund(bg, id)
	}
	s.broadcast(showtimeID, models.SeatStatusAvailable, released)

	return &dto.ShowtimeCancelResponse{
		ShowtimeID:       showtimeID,
		Status:           models.ShowtimeCancelled,
		BookingsAffected: affected,
	}, nil
}

func (s *bookingService) SweepExpired(ctx context.Context, limit int) (SweepResult, error) {
	var res SweepResult

	released, err := s.repo.SweepExpiredHolds(ctx, limit)
	if err != nil {
		return res, err
	}
	res.ReleasedSeats = len(released)
	byShow := make(map[string][]string)
	for _, seat := range released {
		byShow[seat.ShowtimeID] = append(byShow[seat.ShowtimeID], seat.ID)
	}
	for showID, ids := range byShow {
		s.broadcast(showID, models.SeatStatusAvailable, ids)
	}

	expired, err := s.repo.ExpireOverdueUnpaid(ctx, limit)
	if err != nil {
		return res, err
	}
	res.ExpiredBookings = int(expired)

	stuck, err := s.repo.StuckPaidIDs(ctx, limit)
	if err != nil {
		return res, err
	}
	for _, id := range stuck {
		if _, err := s.finalize(ctx, id, sourceSweep); err != nil {
			attempts, derr := s.repo.DeferFinalize(ctx, id)
			logger.Warn("finalize stuck paid booking failed; will retry", logger.String("booking_id", id),
				logger.String("attempt", fmt.Sprint(attempts)), logger.Err(err))
			if derr != nil {
				logger.Warn("schedule finalize retry failed", logger.String("booking_id", id), logger.Err(derr))
			}
			if attempts == stuckAlertAttempts {
				logger.Error("paid booking can not be settled, needs a look", logger.String("booking_id", id))
			}
			continue
		}
		res.FinalizedPaid++
	}

	refunds, err := s.payments.RefundPending(ctx, limit)
	if err != nil {
		return res, err
	}
	for i := range refunds {
		if s.settleRefund(ctx, refunds[i].ID) {
			res.SettledRefunds++
		}
	}

	due, err := s.payments.DueForReconcile(ctx, limit)
	if err != nil {
		return res, err
	}
	for i := range due {
		result, err := s.reconcilePayment(ctx, &due[i])
		if err != nil {
			logger.Warn("reconcile payment failed", logger.String("payment_id", due[i].ID), logger.Err(err))
			continue
		}
		switch result {
		case reconcileApplied:
			res.ReconciledPayments++
		case reconcileAbandoned:
			res.AbandonedPayments++
		}
	}

	late, err := s.payments.FailedForRecheck(ctx, s.lateCaptureWindow, limit)
	if err != nil {
		return res, err
	}
	for i := range late {
		result, err := s.reconcilePayment(ctx, &late[i])
		if err != nil {
			logger.Warn("recheck failed payment failed", logger.String("payment_id", late[i].ID), logger.Err(err))
			continue
		}
		if result == reconcileApplied {
			res.LateCaptures++
		}
	}
	return res, nil
}

// publishTicketEmail failures only log: the sendTicketEmails cron sends whatever is still unsent.
func (s *bookingService) publishTicketEmail(ctx context.Context, bookingID string) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(map[string]string{"booking_id": bookingID})
	if err != nil {
		return
	}
	// A broker that does not confirm must not hold the confirm request.
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	if err := s.publisher.Publish(ctx, TicketEmailQueue, body); err != nil {
		logger.Warn("ticket email enqueue failed; cron will send it",
			logger.String("booking_id", bookingID), logger.Err(err))
	}
}

func transactionOf(p *models.Payment) payment.Transaction {
	t := payment.Transaction{
		TxnRef:        p.TxnRef,
		Amount:        p.Amount,
		ProviderTxnID: derefString(p.ProviderTxnID),
		CreatedAt:     p.CreatedAt,
		PaidAt:        p.PaidAt,
		Data:          p.ProviderData,
	}
	if p.PaidAmount != nil {
		t.PaidAmount = *p.PaidAmount
	}
	return t
}

// newTxnRef: 24 alphanumeric chars, accepted by common Vietnamese gateways' reference rules.
func newTxnRef() string {
	return "CP" + time.Now().UTC().Format("060102150405") + randomHex(5)
}
