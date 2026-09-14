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

// abandonAfter is how long past the booking hold an unpaid attempt is still
// queried before it is given up as abandoned.
const abandonAfter = 15 * time.Minute

// ---------------------------------------------------------------------------
// Pay (F8)
// ---------------------------------------------------------------------------

// Pay opens a checkout with the chosen provider for a pending, unexpired
// booking. The amount comes from the booking (R-P4). Paying again with the
// same provider returns the open checkout (E-P11); another provider opens a
// new attempt. The attempt row is stored before the provider is called, so an
// immediate IPN finds it; the provider is never called inside a transaction.
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
		booking models.Booking
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		attempt, reused = nil, false
		// Account row before booking row, the same order as a hold, so an
		// account lock waiting between them can not deadlock the two.
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
			return apperrors.ErrAccountLocked // E-U3: no new purchase
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
			return apperrors.ErrBookingExpired // E-P7
		}
		booking = *b

		open, err := s.payments.LockOpen(ctx, tx, b.ID, name)
		if err != nil {
			return err
		}
		if open != nil {
			attempt, reused = open, true
			return nil
		}
		attempt = &models.Payment{
			BookingID: b.ID,
			Provider:  name,
			TxnRef:    newTxnRef(),
			Amount:    b.TotalAmount,
			Status:    models.PaymentPending,
			ExpiresAt: b.ExpiresAt,
		}
		if err := s.payments.Create(ctx, tx, attempt); err != nil {
			return err
		}
		return s.audit(ctx, tx, "orders.pay", "payment", attempt.ID, map[string]any{
			"booking_id": b.ID, "provider": name, "txn_ref": attempt.TxnRef, "amount": attempt.Amount,
		})
	})
	if err != nil {
		return nil, err
	}
	if reused {
		if attempt.RedirectURL == nil {
			return nil, apperrors.Conflict("payment is still being created, retry shortly")
		}
		return payResponse(attempt, *attempt.RedirectURL), nil
	}

	checkout, err := provider.CreatePayment(ctx, payment.CreateRequest{
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
	bg := context.WithoutCancel(ctx)
	if err != nil {
		createErr := err
		if ferr := s.inTx(bg, func(tx *gorm.DB) error {
			if _, err := s.payments.MarkFailed(bg, tx, attempt.ID, models.PaymentReasonCreateFailed); err != nil {
				return err
			}
			return s.audit(bg, tx, "payments.create_failed", "payment", attempt.ID,
				map[string]any{"booking_id": booking.ID, "provider": name, "error": createErr.Error()})
		}); ferr != nil {
			logger.Warn("failed checkout not recorded", logger.String("payment_id", attempt.ID), logger.Err(ferr))
		}
		return nil, apperrors.ErrPaymentGateway.Wrap(createErr) // E-P10: booking untouched, pay again later
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

// callbackURL is the public IPN or return URL of a provider.
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

// ---------------------------------------------------------------------------
// Provider callbacks
// ---------------------------------------------------------------------------

// HandleNotification processes an IPN of any provider. A valid "paid" notice
// means money was collected: the booking is confirmed or refunded right away.
// Duplicates are no-ops (E-P2); a database failure asks the provider to retry
// (E-P12).
func (s *bookingService) HandleNotification(ctx context.Context, provider payment.Provider, r *http.Request) payment.AckStatus {
	n, err := provider.ParseNotification(r)
	switch {
	case errors.Is(err, payment.ErrInvalidSignature):
		s.auditRejected(ctx, provider.Name(), "", "invalid signature") // E-P1
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

// HandleReturn processes the browser coming back from a provider. The redirect
// only tells which attempt it was; the real state is asked from the provider.
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

// applyNotification applies a verified provider notice (IPN or query result)
// under the attempt row lock.
func (s *bookingService) applyNotification(ctx context.Context, providerName string, n *payment.Notification, source string) (payment.AckStatus, error) {
	var (
		ack       payment.AckStatus
		out       finalizeOutcome
		bookingID string
		showID    string
	)
	err := s.inTx(ctx, func(tx *gorm.DB) error {
		ack, out, bookingID, showID = payment.AckProcessed, finalizeOutcome{}, "", ""

		attempt, err := s.payments.LockByRef(ctx, tx, providerName, n.TxnRef)
		if err != nil {
			return err
		}
		if attempt == nil {
			ack = payment.AckUnknownTxn
			return nil
		}
		bookingID = attempt.BookingID
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
			return s.audit(ctx, tx, "payments.failed", "payment", attempt.ID,
				map[string]any{"booking_id": attempt.BookingID, "provider": providerName, "source": source})
		case payment.StatePaid:
		default:
			ack = payment.AckInvalid
			return nil
		}

		b, err := s.repo.LockBooking(ctx, tx, attempt.BookingID)
		if err != nil {
			return err
		}
		if b == nil {
			return fmt.Errorf("payment %s: booking %s missing", attempt.ID, attempt.BookingID)
		}
		showID = b.ShowtimeID

		// The booking already carries money of another attempt (the customer
		// paid twice, e.g. with two providers): this payment goes straight back.
		if b.PaidAt != nil {
			if _, err := s.payments.MarkCaptured(ctx, tx, attempt.ID, models.PaymentRefundPending,
				n.Amount, n.ProviderTxnID, n.Data, models.PaymentReasonDuplicate); err != nil {
				return err
			}
			out.refunds = append(out.refunds, attempt.ID)
			return s.audit(ctx, tx, "orders.refund", "payment", attempt.ID, map[string]any{
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
			reason = models.ReasonAmountMismatch // E-P4
		}
		return s.finalizeTx(ctx, tx, b, reason, source, &out)
	})
	if err != nil {
		return 0, err
	}
	s.afterFinalize(ctx, bookingID, showID, out)
	return ack, nil
}

// ---------------------------------------------------------------------------
// Confirm / refund (F9)
// ---------------------------------------------------------------------------

// Confirm lets the customer (or the frontend after returning from the
// checkout) settle a paid booking. It reconciles with the provider first, so
// a lost IPN does not block the ticket (E-P5).
func (s *bookingService) Confirm(ctx context.Context, userID, bookingID string) (*dto.OrderDetailResponse, error) {
	b, err := s.ownedBooking(ctx, userID, bookingID)
	if err != nil {
		return nil, err
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

// finalizeOutcome is what one finalize transaction did.
type finalizeOutcome struct {
	confirmed bool
	sold      []string
	released  []string
	refunds   []string // payment ids to refund at the provider after commit
}

// finalize settles a paid booking found by id (customer confirm, crash
// recovery). Confirmed and refunded bookings are returned untouched.
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

// finalizeTx is the only place a paid booking leaves PENDING. With the booking
// locked and its payment recorded, in ONE transaction it either sells every
// held seat (fenced by hold version, TTL and showtime state) and issues the
// tickets, or releases what is left and marks booking REFUNDED + payment
// REFUND_PENDING. The provider refund runs after commit (R-HO3). Because the
// booking row is locked, concurrent IPN / reconcile / confirm converge on one
// outcome (E-C3). A non-empty reason forces the refund.
func (s *bookingService) finalizeTx(ctx context.Context, tx *gorm.DB, b *models.Booking, reason, source string, out *finalizeOutcome) error {
	switch b.Status {
	case models.BookingConfirmed, models.BookingRefunded:
		return nil
	case models.BookingExpired:
		if reason == "" {
			reason = models.ReasonPaidAfterExpiry // E-P3
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
			reason = models.ReasonShowtimeClosed // E-C5
		case b.ExpiresAt == nil || !b.ExpiresAt.After(now):
			reason = models.ReasonHoldExpired // E-C1, E-C4
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
				reason = models.ReasonSeatsLost // E-C2: swept or taken over
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
	return s.audit(ctx, tx, "orders.confirm", "booking", b.ID, map[string]any{
		"status": models.BookingConfirmed, "tickets": len(tickets), "total": total,
		"payment_id": derefString(b.PaymentID), "source": source,
	})
}

// refundTx releases the seats still held by the booking, marks it REFUNDED
// and its payment REFUND_PENDING, in the caller's transaction.
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
	return s.audit(ctx, tx, "orders.refund", "booking", b.ID, map[string]any{
		"status": models.BookingRefunded, "reason": reason, "amount": b.TotalAmount,
		"payment_id": *b.PaymentID, "source": source,
	})
}

// afterFinalize runs the side effects of a committed finalize. They must not
// die with the request context.
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

// settleRefund asks the provider to return the money of a REFUND_PENDING
// attempt. A failure leaves it pending; the sweep retries.
func (s *bookingService) settleRefund(ctx context.Context, paymentID string) bool {
	attempt, err := s.payments.FindByID(ctx, paymentID)
	if err != nil || attempt == nil || attempt.Status != models.PaymentRefundPending {
		return false
	}
	provider, ok := s.providers.Get(attempt.Provider)
	if !ok {
		logger.Warn("refund waits: payment provider not enabled",
			logger.String("payment_id", attempt.ID), logger.String("provider", attempt.Provider))
		return false
	}
	amount := attempt.Amount
	if attempt.PaidAmount != nil {
		amount = *attempt.PaidAmount
	}
	err = provider.Refund(ctx, transactionOf(attempt), payment.RefundRequest{Amount: amount, Reason: derefString(attempt.StatusReason)})
	if err != nil {
		logger.Warn("provider refund failed; sweep will retry", logger.String("payment_id", attempt.ID), logger.Err(err))
		return false
	}
	err = s.inTx(ctx, func(tx *gorm.DB) error {
		n, err := s.payments.MarkRefunded(ctx, tx, attempt.ID)
		if err != nil || n == 0 {
			return err
		}
		return s.audit(ctx, tx, "payments.refunded", "payment", attempt.ID, map[string]any{
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

// ---------------------------------------------------------------------------
// Reconcile
// ---------------------------------------------------------------------------

// reconcile pulls the real payment state of a booking's open attempts from
// their providers (the IPN may be lost) and settles a paid booking whose
// confirm never completed.
func (s *bookingService) reconcile(ctx context.Context, b *models.Booking) error {
	if b.Status == models.BookingPending && b.PaidAt != nil {
		_, err := s.finalize(ctx, b.ID, sourceReconcile)
		return err
	}
	if b.Status != models.BookingPending && b.Status != models.BookingExpired {
		return nil
	}
	attempts, err := s.payments.OpenForBooking(ctx, b.ID)
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

// reconcilePayment queries one open attempt and applies what the provider
// reports; an attempt still unpaid long after its hold is marked abandoned.
func (s *bookingService) reconcilePayment(ctx context.Context, attempt *models.Payment) (reconcileResult, error) {
	if attempt.Status != models.PaymentPending {
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
			return s.audit(ctx, tx, "payments.abandoned", "payment", attempt.ID,
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

// ---------------------------------------------------------------------------
// Sweep (F10)
// ---------------------------------------------------------------------------

// SweepExpired is one pass of the sweepExpiredHolds job, each step bounded by
// limit: release expired seat holds, expire unpaid overdue bookings, finalize
// paid bookings stuck in PENDING, retry provider refunds, reconcile open
// payment attempts whose IPN may be lost.
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
			logger.Warn("finalize stuck paid booking failed", logger.String("booking_id", id), logger.Err(err))
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
	return res, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// publishTicketEmail enqueues the ticket email; failures only log because the
// sendTicketEmails cron sends whatever is still unsent (R-ML1).
func (s *bookingService) publishTicketEmail(ctx context.Context, bookingID string) {
	if s.publisher == nil {
		return
	}
	body, err := json.Marshal(map[string]string{"booking_id": bookingID})
	if err != nil {
		return
	}
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

// newTxnRef is the merchant reference sent to providers: 24 alphanumeric
// chars, accepted by the common Vietnamese gateways' reference rules.
func newTxnRef() string {
	return "CP" + time.Now().UTC().Format("060102150405") + randomHex(5)
}
