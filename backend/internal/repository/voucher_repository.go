package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type VoucherRepository struct {
	db *gorm.DB
}

func NewVoucherRepository(db *gorm.DB) *VoucherRepository {
	return &VoucherRepository{db: db}
}

func (r *VoucherRepository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}

func (r *VoucherRepository) Create(tx *gorm.DB, v *models.Voucher) error {
	return tx.Create(v).Error
}

func (r *VoucherRepository) List(ctx context.Context, status string, limit, offset int) ([]models.Voucher, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Voucher{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vouchers []models.Voucher
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&vouchers).Error
	return vouchers, total, err
}

func (r *VoucherRepository) FindByID(ctx context.Context, id string) (*models.Voucher, error) {
	var v models.Voucher
	if err := r.db.WithContext(ctx).First(&v, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

// LockByCode locks the voucher row for the duration of the hold transaction,
// matching the lock order advisory -> user -> showtime -> seats -> voucher
// (ADVANCED_FEATURES_DISCUSSION.md Phần 1.1).
func (r *VoucherRepository) LockByCode(tx *gorm.DB, code string) (*models.Voucher, error) {
	var v models.Voucher
	err := tx.Clauses(forUpdate()).First(&v, "code = ?", code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *VoucherRepository) Update(tx *gorm.DB, v *models.Voucher) error {
	return tx.Model(v).Select(
		"discount_type", "discount_value", "max_discount", "min_order_amount",
		"starts_at", "ends_at", "max_usage", "max_usage_per_user", "apply_scope",
		"branch_id", "campaign_id", "updated_at",
	).Updates(v).Error
}

func (r *VoucherRepository) SetStatus(tx *gorm.DB, id, status string) (int64, error) {
	res := tx.Model(&models.Voucher{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": gorm.Expr("NOW()")})
	return res.RowsAffected, res.Error
}

// IncrUsage is the CAS that reserves one use: it only succeeds while
// usage_count is still below max_usage, so a hotspot voucher never oversells
// even without the Redis pre-filter layer.
func (r *VoucherRepository) IncrUsage(tx *gorm.DB, id string) (int64, error) {
	res := tx.Exec(`UPDATE vouchers SET usage_count = usage_count + 1, updated_at = NOW()
		WHERE id = ? AND usage_count < max_usage`, id)
	if res.Error != nil {
		return 0, fmt.Errorf("incr voucher usage: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// DecrUsage releases one use back to the pool (booking expired/canceled/refunded
// before or without ever being confirmed).
func (r *VoucherRepository) DecrUsage(tx *gorm.DB, id string) error {
	return tx.Exec(`UPDATE vouchers SET usage_count = GREATEST(usage_count - 1, 0), updated_at = NOW()
		WHERE id = ?`, id).Error
}

func (r *VoucherRepository) CountUserRedemptions(tx *gorm.DB, voucherID, userID string) (int64, error) {
	var n int64
	err := tx.Model(&models.VoucherRedemption{}).
		Where("voucher_id = ? AND user_id = ?", voucherID, userID).Count(&n).Error
	return n, err
}

func (r *VoucherRepository) CreateRedemption(tx *gorm.DB, red *models.VoucherRedemption) error {
	return tx.Create(red).Error
}

func (r *VoucherRepository) FindRedemptionByBooking(ctx context.Context, tx *gorm.DB, bookingID string) (*models.VoucherRedemption, error) {
	var red models.VoucherRedemption
	if err := r.conn(ctx, tx).First(&red, "booking_id = ?", bookingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &red, nil
}

func (r *VoucherRepository) DeleteRedemption(tx *gorm.DB, bookingID string) error {
	return tx.Where("booking_id = ?", bookingID).Delete(&models.VoucherRedemption{}).Error
}

// ReleaseUsage is the rollback pair to IncrUsage+CreateRedemption: called
// whenever a booking that consumed a voucher does not end up CONFIRMED
// (replaced by a new hold, canceled, expired, or refunded — Phần 1.1
// "Refund kỹ thuật (F9) phải rollback voucher").
// VoucherConflictRow is a redemption where the redeeming user is also the
// voucher's own creator (Phần 9.4 conflict-of-interest detection).
type VoucherConflictRow struct {
	VoucherID string
	Code      string
	AdminID   string
	BookingID string
	UsedAt    time.Time
}

func (r *VoucherRepository) Conflicts(ctx context.Context) ([]VoucherConflictRow, error) {
	var rows []VoucherConflictRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT v.id AS voucher_id, v.code, v.created_by_admin_id AS admin_id, vr.booking_id, vr.used_at
		FROM voucher_redemptions vr
		JOIN vouchers v ON v.id = vr.voucher_id
		WHERE vr.user_id = v.created_by_admin_id
		ORDER BY vr.used_at DESC`).Scan(&rows).Error
	return rows, err
}

func (r *VoucherRepository) ReleaseUsage(ctx context.Context, tx *gorm.DB, bookingID string) error {
	red, err := r.FindRedemptionByBooking(ctx, tx, bookingID)
	if err != nil || red == nil {
		return err
	}
	if err := r.DecrUsage(tx, red.VoucherID); err != nil {
		return err
	}
	return r.DeleteRedemption(tx, bookingID)
}
