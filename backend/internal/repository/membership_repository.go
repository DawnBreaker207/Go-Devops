package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type MembershipRepository struct {
	db *gorm.DB
}

func NewMembershipRepository(db *gorm.DB) *MembershipRepository {
	return &MembershipRepository{db: db}
}

func (r *MembershipRepository) CreateTier(tx *gorm.DB, t *models.MembershipTier) error {
	return tx.Create(t).Error
}

func (r *MembershipRepository) ListTiers(ctx context.Context, activeOnly bool) ([]models.MembershipTier, error) {
	q := r.db.WithContext(ctx).Order("price")
	if activeOnly {
		q = q.Where("active = true")
	}
	var tiers []models.MembershipTier
	err := q.Find(&tiers).Error
	return tiers, err
}

func (r *MembershipRepository) FindTierByID(ctx context.Context, id string) (*models.MembershipTier, error) {
	var t models.MembershipTier
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *MembershipRepository) UpdateTier(tx *gorm.DB, t *models.MembershipTier) error {
	return tx.Model(t).Select(
		"name", "price", "duration_days", "discount_percent", "excluded_seat_types", "active", "updated_at",
	).Updates(t).Error
}

// LockActiveByUser locks the caller's own active-membership slot for the
// duration of a purchase/renew transaction (one_active_membership_per_user).
func (r *MembershipRepository) LockActiveByUser(tx *gorm.DB, userID string) (*models.UserMembership, error) {
	var m models.UserMembership
	err := tx.Clauses(forUpdate()).First(&m, "user_id = ? AND status = ?", userID, models.MembershipActive).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MembershipRepository) FindByIdempotencyKey(tx *gorm.DB, key string) (*models.UserMembership, error) {
	var m models.UserMembership
	err := tx.First(&m, "idempotency_key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MembershipRepository) Create(tx *gorm.DB, m *models.UserMembership) error {
	return tx.Create(m).Error
}

func (r *MembershipRepository) ExtendExpiry(tx *gorm.DB, id string, expiresAt time.Time) error {
	return tx.Model(&models.UserMembership{}).Where("id = ?", id).
		Updates(map[string]any{"expires_at": expiresAt, "updated_at": gorm.Expr("NOW()")}).Error
}

func (r *MembershipRepository) ExpirePast(tx *gorm.DB) (int64, error) {
	res := tx.Exec(`UPDATE user_memberships SET status = ?, updated_at = NOW()
		WHERE status = ? AND expires_at < NOW()`, models.MembershipExpired, models.MembershipActive)
	return res.RowsAffected, res.Error
}

// ActiveForUser is a plain (unlocked) read used to price a booking at hold
// time: the discount is a snapshot, consistent with how seat prices are
// already chosen once and baked in (ADVANCED_FEATURES_DISCUSSION.md Phần 1.2).
func (r *MembershipRepository) ActiveForUser(ctx context.Context, tx *gorm.DB, userID string) (*models.UserMembership, *models.MembershipTier, error) {
	conn := r.db.WithContext(ctx)
	if tx != nil {
		conn = tx
	}
	var m models.UserMembership
	err := conn.Where("user_id = ? AND status = ? AND expires_at > NOW()", userID, models.MembershipActive).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var tier models.MembershipTier
	if err := conn.First(&tier, "id = ?", m.TierID).Error; err != nil {
		return nil, nil, err
	}
	return &m, &tier, nil
}

func (r *MembershipRepository) ListByUser(ctx context.Context, userID string) ([]models.UserMembership, error) {
	var rows []models.UserMembership
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

type UserMembershipRow struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// ExpiringSoon: active memberships expiring within [from, to) that have not
// been reminded yet, for the renewal-reminder batch job.
func (r *MembershipRepository) ExpiringSoon(ctx context.Context, from, to time.Time) ([]UserMembershipRow, error) {
	var rows []UserMembershipRow
	err := r.db.WithContext(ctx).Model(&models.UserMembership{}).
		Where("status = ? AND expires_at >= ? AND expires_at < ? AND reminder_sent_at IS NULL", models.MembershipActive, from, to).
		Find(&rows).Error
	return rows, err
}

func (r *MembershipRepository) MarkReminderSent(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Exec(`UPDATE user_memberships SET reminder_sent_at = NOW() WHERE id = ?`, id).Error
}
