package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WaitlistRepository struct {
	db *gorm.DB
}

func NewWaitlistRepository(db *gorm.DB) *WaitlistRepository {
	return &WaitlistRepository{db: db}
}

// AvailableSeatCount backs the join-time check ("chỉ đăng ký chờ khi suất
// hết ghế"): non-gap seats currently available or holdable (an expired hold
// still counts as taken until the sweep frees it, same as everywhere else).
func (r *WaitlistRepository) AvailableSeatCount(ctx context.Context, showtimeID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM showtime_seats ss JOIN seats s ON s.id = ss.seat_id
		WHERE ss.showtime_id = ? AND s.is_gap = false
		  AND (ss.status = 'available' OR (ss.status = 'held' AND ss.held_until < NOW()))`,
		showtimeID).Scan(&n).Error
	return n, err
}

func (r *WaitlistRepository) CountActiveByUser(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.WaitlistEntry{}).
		Where("user_id = ? AND status IN ?", userID, []string{models.WaitlistWaiting, models.WaitlistNotified}).Count(&n).Error
	return n, err
}

func (r *WaitlistRepository) Create(tx *gorm.DB, e *models.WaitlistEntry) error {
	return tx.Create(e).Error
}

func (r *WaitlistRepository) FindByID(ctx context.Context, id string) (*models.WaitlistEntry, error) {
	var e models.WaitlistEntry
	if err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *WaitlistRepository) MyEntries(ctx context.Context, userID string) ([]models.WaitlistEntry, error) {
	var rows []models.WaitlistEntry
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *WaitlistRepository) Cancel(tx *gorm.DB, id, userID string) (int64, error) {
	res := tx.Model(&models.WaitlistEntry{}).
		Where("id = ? AND user_id = ? AND status IN ?", id, userID, []string{models.WaitlistWaiting, models.WaitlistNotified}).
		Update("status", models.WaitlistCanceled)
	return res.RowsAffected, res.Error
}

// NextWaiting locks (SKIP LOCKED) the oldest 'waiting' entry of a showtime,
// FIFO — used by the sweep to decide who a just-released seat goes to.
func (r *WaitlistRepository) NextWaiting(tx *gorm.DB, showtimeID string) (*models.WaitlistEntry, error) {
	var e models.WaitlistEntry
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("showtime_id = ? AND status = ?", showtimeID, models.WaitlistWaiting).
		Order("created_at").Limit(1).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("lock next waitlist entry: %w", err)
	}
	return &e, nil
}

func (r *WaitlistRepository) MarkNotified(tx *gorm.DB, id, bookingID string, expiresAt time.Time) error {
	return tx.Model(&models.WaitlistEntry{}).Where("id = ?", id).Updates(map[string]any{
		"status": models.WaitlistNotified, "fulfilled_booking_id": bookingID,
		"notified_at": gorm.Expr("NOW()"), "expires_at": expiresAt,
	}).Error
}

// ExpireStale flips a 'notified' entry to 'expired' once its hold window has
// passed without the customer paying — called by the same sweep that
// expires ordinary holds, so the next waitlist entry (if any) is not stuck
// behind someone who never acted. The seat itself is already released by
// the normal expired-hold path; this only updates the waitlist bookkeeping.
func (r *WaitlistRepository) ExpireStale(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`UPDATE waitlist_entries SET status = ?
		WHERE status = ? AND expires_at < NOW()
		AND id IN (SELECT id FROM waitlist_entries WHERE status = ? AND expires_at < NOW() LIMIT ?)`,
		models.WaitlistExpired, models.WaitlistNotified, models.WaitlistNotified, limit)
	return res.RowsAffected, res.Error
}

// CancelForClosedShowtime cancels every still-active entry of a showtime
// that was closed/deleted while people were waiting.
func (r *WaitlistRepository) CancelForClosedShowtime(tx *gorm.DB, showtimeID string) error {
	return tx.Model(&models.WaitlistEntry{}).
		Where("showtime_id = ? AND status IN ?", showtimeID, []string{models.WaitlistWaiting, models.WaitlistNotified}).
		Update("status", models.WaitlistCanceled).Error
}
