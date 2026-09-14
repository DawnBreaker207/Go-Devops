package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// PaymentRepository stores payment attempts, whatever the provider.
type PaymentRepository interface {
	Create(ctx context.Context, tx *gorm.DB, p *models.Payment) error
	FindByID(ctx context.Context, id string) (*models.Payment, error)
	FindByRef(ctx context.Context, provider, txnRef string) (*models.Payment, error)
	LockByRef(ctx context.Context, tx *gorm.DB, provider, txnRef string) (*models.Payment, error)
	LockOpen(ctx context.Context, tx *gorm.DB, bookingID, provider string) (*models.Payment, error)
	HasOpen(ctx context.Context, tx *gorm.DB, bookingID string) (bool, error)
	ReconcilableForBooking(ctx context.Context, bookingID string, lateWindow time.Duration) ([]models.Payment, error)
	LatestForBooking(ctx context.Context, bookingID string) (*models.Payment, error)

	ClaimOrphanCheckout(ctx context.Context, tx *gorm.DB, id string, after time.Duration) (int64, error)
	SetCheckout(ctx context.Context, id, redirectURL, providerTxnID string, data map[string]string) error
	MarkFailed(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	MarkCaptured(ctx context.Context, tx *gorm.DB, id, status string, amount int64, providerTxnID string, data map[string]string, reason string) (int64, error)
	MarkRefundPending(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	MarkRefunded(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	TouchChecked(ctx context.Context, id string) error

	ClaimRefund(ctx context.Context, id string, lease time.Duration) (*models.Payment, error)
	DeferRefund(ctx context.Context, id, lastError string) error

	RefundPending(ctx context.Context, limit int) ([]models.Payment, error)
	DueForReconcile(ctx context.Context, limit int) ([]models.Payment, error)
	FailedForRecheck(ctx context.Context, lateWindow time.Duration, limit int) ([]models.Payment, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}

func jsonObject(data map[string]string) string {
	if len(data) == 0 {
		return "{}"
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (r *paymentRepository) Create(ctx context.Context, tx *gorm.DB, p *models.Payment) error {
	if err := r.conn(ctx, tx).Create(p).Error; err != nil {
		return fmt.Errorf("create payment: %w", err)
	}
	return nil
}

func (r *paymentRepository) FindByID(ctx context.Context, id string) (*models.Payment, error) {
	return firstOrNil[models.Payment](r.db.WithContext(ctx).Where("id = ?", id), "find payment")
}

func (r *paymentRepository) FindByRef(ctx context.Context, provider, txnRef string) (*models.Payment, error) {
	return firstOrNil[models.Payment](r.db.WithContext(ctx).Where("provider = ? AND txn_ref = ?", provider, txnRef), "find payment by ref")
}

// LockByRef serializes every notification/reconcile of one attempt.
func (r *paymentRepository) LockByRef(ctx context.Context, tx *gorm.DB, provider, txnRef string) (*models.Payment, error) {
	return firstOrNil[models.Payment](r.conn(ctx, tx).Clauses(forUpdate()).
		Where("provider = ? AND txn_ref = ?", provider, txnRef), "lock payment by ref")
}

func (r *paymentRepository) LockOpen(ctx context.Context, tx *gorm.DB, bookingID, provider string) (*models.Payment, error) {
	return firstOrNil[models.Payment](r.conn(ctx, tx).Clauses(forUpdate()).
		Where("booking_id = ? AND provider = ? AND status = ?", bookingID, provider, models.PaymentPending), "lock open payment")
}

// HasOpen reports a checkout the customer may be paying right now. An attempt
// that never got its checkout URL (orphan) stops counting after a short grace
// period, so it can not block a new hold until the booking expires.
func (r *paymentRepository) HasOpen(ctx context.Context, tx *gorm.DB, bookingID string) (bool, error) {
	var n int64
	if err := r.conn(ctx, tx).Model(&models.Payment{}).
		Where("booking_id = ? AND status = ?", bookingID, models.PaymentPending).
		Where("redirect_url IS NOT NULL OR updated_at > NOW() - make_interval(secs => ?)", orphanGrace.Seconds()).
		Count(&n).Error; err != nil {
		return false, fmt.Errorf("count open payments: %w", err)
	}
	return n > 0, nil
}

// orphanGrace is how long a stored attempt may wait for its checkout URL
// before it counts as orphaned.
const orphanGrace = 30 * time.Second

// ReconcilableForBooking lists the attempts of a booking worth asking the
// provider about: every open attempt, plus given-up ones still inside the late
// capture window that were not queried in the last couple of minutes.
func (r *paymentRepository) ReconcilableForBooking(ctx context.Context, bookingID string, lateWindow time.Duration) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).
		Where(`status = ? OR (status = ? AND COALESCE(expires_at, created_at) > NOW() - make_interval(secs => ?)
			AND (checked_at IS NULL OR checked_at < NOW() - INTERVAL '2 minutes'))`,
			models.PaymentPending, models.PaymentFailed, lateWindow.Seconds()).
		Order("created_at").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find reconcilable payments: %w", err)
	}
	return out, nil
}

// ClaimOrphanCheckout takes over an open attempt that never got its checkout
// URL for longer than after; 0 rows means it is too recent or already taken.
func (r *paymentRepository) ClaimOrphanCheckout(ctx context.Context, tx *gorm.DB, id string, after time.Duration) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE payments SET updated_at = NOW()
		WHERE id = ? AND status = ? AND redirect_url IS NULL AND updated_at < NOW() - make_interval(secs => ?)`,
		id, models.PaymentPending, after.Seconds())
	if res.Error != nil {
		return 0, fmt.Errorf("claim orphan checkout: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *paymentRepository) LatestForBooking(ctx context.Context, bookingID string) (*models.Payment, error) {
	return firstOrNil[models.Payment](r.db.WithContext(ctx).Where("booking_id = ?", bookingID).Order("created_at DESC"), "find latest payment")
}

func (r *paymentRepository) SetCheckout(ctx context.Context, id, redirectURL, providerTxnID string, data map[string]string) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE payments SET redirect_url = ?,
			provider_txn_id = COALESCE(NULLIF(?, ''), provider_txn_id),
			provider_data = provider_data || ?::jsonb, updated_at = NOW()
		WHERE id = ?`, redirectURL, providerTxnID, jsonObject(data), id).Error; err != nil {
		return fmt.Errorf("set payment checkout: %w", err)
	}
	return nil
}

func (r *paymentRepository) MarkFailed(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE payments SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status = ?`, models.PaymentFailed, reason, id, models.PaymentPending)
	if res.Error != nil {
		return 0, fmt.Errorf("mark payment failed: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// MarkCaptured records money the provider collected. A failed attempt can
// still be captured: a gateway may settle after we gave up on it.
func (r *paymentRepository) MarkCaptured(ctx context.Context, tx *gorm.DB, id, status string, amount int64, providerTxnID string, data map[string]string, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE payments SET status = ?, status_reason = NULLIF(?, ''),
			paid_amount = ?, paid_at = NOW(),
			provider_txn_id = COALESCE(NULLIF(?, ''), provider_txn_id),
			provider_data = provider_data || ?::jsonb, updated_at = NOW()
		WHERE id = ? AND status IN (?, ?)`,
		status, reason, amount, providerTxnID, jsonObject(data), id, models.PaymentPending, models.PaymentFailed)
	if res.Error != nil {
		return 0, fmt.Errorf("mark payment captured: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *paymentRepository) MarkRefundPending(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE payments SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status = ?`, models.PaymentRefundPending, reason, id, models.PaymentPaid)
	if res.Error != nil {
		return 0, fmt.Errorf("mark payment refund pending: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *paymentRepository) MarkRefunded(ctx context.Context, tx *gorm.DB, id string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE payments SET status = ?, refunded_at = NOW(), updated_at = NOW()
		WHERE id = ? AND status = ?`, models.PaymentRefunded, id, models.PaymentRefundPending)
	if res.Error != nil {
		return 0, fmt.Errorf("mark payment refunded: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *paymentRepository) TouchChecked(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE payments SET checked_at = NOW() WHERE id = ?`, id).Error; err != nil {
		return fmt.Errorf("touch payment checked_at: %w", err)
	}
	return nil
}

// ClaimRefund reserves a REFUND_PENDING attempt that is due for the lease time
// before its provider is called, and counts the try. nil means nothing to do:
// settled already, claimed by another worker, or waiting for its retry time.
func (r *paymentRepository) ClaimRefund(ctx context.Context, id string, lease time.Duration) (*models.Payment, error) {
	res := r.db.WithContext(ctx).Exec(`UPDATE payments
		SET next_retry_at = NOW() + make_interval(secs => ?), refund_attempts = refund_attempts + 1, updated_at = NOW()
		WHERE id = ? AND status = ? AND (next_retry_at IS NULL OR next_retry_at <= NOW())`,
		lease.Seconds(), id, models.PaymentRefundPending)
	if res.Error != nil {
		return nil, fmt.Errorf("claim refund: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return r.FindByID(ctx, id)
}

// DeferRefund schedules the next try of a failed refund: 1 minute doubled on
// every failure, at most 1 hour.
func (r *paymentRepository) DeferRefund(ctx context.Context, id, lastError string) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE payments
		SET next_retry_at = NOW() + LEAST(INTERVAL '1 hour', INTERVAL '1 minute' * power(2, GREATEST(refund_attempts - 1, 0))),
			last_error = left(?, 512), updated_at = NOW()
		WHERE id = ? AND status = ?`, lastError, id, models.PaymentRefundPending).Error; err != nil {
		return fmt.Errorf("defer refund: %w", err)
	}
	return nil
}

// RefundPending lists refunds due for a try, the longest waiting first, so a
// refund failing for good never starves newer ones.
func (r *paymentRepository) RefundPending(ctx context.Context, limit int) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).Where("status = ?", models.PaymentRefundPending).
		Where("next_retry_at IS NULL OR next_retry_at <= NOW()").
		Order("next_retry_at NULLS FIRST, updated_at").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find pending refunds: %w", err)
	}
	return out, nil
}

// FailedForRecheck lists given-up attempts still inside the late capture
// window, not queried in the last 10 minutes: a gateway may collect the money
// after we stopped waiting, and it must then be confirmed or refunded (E-P3).
func (r *paymentRepository) FailedForRecheck(ctx context.Context, lateWindow time.Duration, limit int) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).Where("status = ?", models.PaymentFailed).
		Where("COALESCE(expires_at, created_at) > NOW() - make_interval(secs => ?)", lateWindow.Seconds()).
		Where("checked_at IS NULL OR checked_at < NOW() - INTERVAL '10 minutes'").
		Order("created_at").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find failed payments to recheck: %w", err)
	}
	return out, nil
}

// DueForReconcile lists open attempts old enough that their IPN may be lost
// and not queried in the last couple of minutes.
func (r *paymentRepository) DueForReconcile(ctx context.Context, limit int) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < NOW() - INTERVAL '2 minutes'", models.PaymentPending).
		Where("checked_at IS NULL OR checked_at < NOW() - INTERVAL '2 minutes'").
		Order("created_at").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find payments to reconcile: %w", err)
	}
	return out, nil
}
