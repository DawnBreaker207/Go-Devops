package repository

import (
	"context"
	"encoding/json"
	"fmt"

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
	OpenForBooking(ctx context.Context, bookingID string) ([]models.Payment, error)
	LatestForBooking(ctx context.Context, bookingID string) (*models.Payment, error)

	SetCheckout(ctx context.Context, id, redirectURL, providerTxnID string, data map[string]string) error
	MarkFailed(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	MarkCaptured(ctx context.Context, tx *gorm.DB, id, status string, amount int64, providerTxnID string, data map[string]string, reason string) (int64, error)
	MarkRefundPending(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	MarkRefunded(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	TouchChecked(ctx context.Context, id string) error

	RefundPending(ctx context.Context, limit int) ([]models.Payment, error)
	DueForReconcile(ctx context.Context, limit int) ([]models.Payment, error)
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

func (r *paymentRepository) HasOpen(ctx context.Context, tx *gorm.DB, bookingID string) (bool, error) {
	var n int64
	if err := r.conn(ctx, tx).Model(&models.Payment{}).
		Where("booking_id = ? AND status = ?", bookingID, models.PaymentPending).Count(&n).Error; err != nil {
		return false, fmt.Errorf("count open payments: %w", err)
	}
	return n > 0, nil
}

func (r *paymentRepository) OpenForBooking(ctx context.Context, bookingID string) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).Where("booking_id = ? AND status = ?", bookingID, models.PaymentPending).
		Order("created_at").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find open payments: %w", err)
	}
	return out, nil
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

func (r *paymentRepository) RefundPending(ctx context.Context, limit int) ([]models.Payment, error) {
	var out []models.Payment
	if err := r.db.WithContext(ctx).Where("status = ?", models.PaymentRefundPending).
		Order("updated_at").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find pending refunds: %w", err)
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
