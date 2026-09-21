package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type NotificationPreferenceRepository interface {
	FindByUserID(ctx context.Context, userID string) (*models.NotificationPreference, error)
	Create(ctx context.Context, tx *gorm.DB, p *models.NotificationPreference) error
	Update(ctx context.Context, tx *gorm.DB, userID string, bookingReminders, promoOffers bool) error
}

type notificationPreferenceRepository struct {
	db *gorm.DB
}

func NewNotificationPreferenceRepository(db *gorm.DB) NotificationPreferenceRepository {
	return &notificationPreferenceRepository{db: db}
}

func (r *notificationPreferenceRepository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}

func (r *notificationPreferenceRepository) FindByUserID(ctx context.Context, userID string) (*models.NotificationPreference, error) {
	return firstOrNil[models.NotificationPreference](r.db.WithContext(ctx).Where("user_id = ?", userID), "find notification preference")
}

func (r *notificationPreferenceRepository) Create(ctx context.Context, tx *gorm.DB, p *models.NotificationPreference) error {
	if err := r.conn(ctx, tx).Create(p).Error; err != nil {
		return fmt.Errorf("create notification preference: %w", err)
	}
	return nil
}

func (r *notificationPreferenceRepository) Update(ctx context.Context, tx *gorm.DB, userID string, bookingReminders, promoOffers bool) error {
	if err := r.conn(ctx, tx).Model(&models.NotificationPreference{}).Where("user_id = ?", userID).
		Updates(map[string]any{"booking_reminders": bookingReminders, "promo_offers": promoOffers}).Error; err != nil {
		return fmt.Errorf("update notification preference: %w", err)
	}
	return nil
}
