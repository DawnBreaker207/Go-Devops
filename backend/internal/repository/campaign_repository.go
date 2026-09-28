package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// CampaignRepository stores campaigns, their combo/article links, and the
// per-account redemption guard on discount codes.
type CampaignRepository interface {
	FindByID(ctx context.Context, id string) (*models.Campaign, error)
	List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.Campaign, int64, error)
	Create(ctx context.Context, tx *gorm.DB, c *models.Campaign) error
	// Update writes only the columns named in `fields`, so a partial request can
	// never blank the columns it did not mention.
	Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error
	Delete(ctx context.Context, tx *gorm.DB, id string) error

	// ListActive/FindActiveByID are the public read path: active=true AND now
	// between starts_at and ends_at.
	ListActive(ctx context.Context, now time.Time, page, pageSize int) ([]models.Campaign, int64, error)
	FindActiveByID(ctx context.Context, id string, now time.Time) (*models.Campaign, error)

	// AttachCombo upserts: re-attaching an already-linked combo updates its
	// promo_price rather than erroring.
	AttachCombo(ctx context.Context, tx *gorm.DB, campaignID, comboID string, promoPrice *int64) error
	DetachCombo(ctx context.Context, tx *gorm.DB, campaignID, comboID string) error
	ListCombos(ctx context.Context, campaignID string) ([]models.CampaignCombo, error)

	// AttachArticle is idempotent (ON CONFLICT DO NOTHING): there is nothing on
	// the link itself worth updating.
	AttachArticle(ctx context.Context, tx *gorm.DB, campaignID, articleID string) error
	DetachArticle(ctx context.Context, tx *gorm.DB, campaignID, articleID string) error
	ListArticles(ctx context.Context, campaignID string) ([]models.CampaignArticle, error)

// ClaimRedemption inserts one-account-one-code guard; caller checks IsUniqueViolation:
// duplicate means same user already redeemed this code.
	ClaimRedemption(ctx context.Context, tx *gorm.DB, userID, discountCodeID string) error
	// ReleaseRedemption gives the redemption back so the same code can be
	// applied again (see DiscountService.Remove).
	ReleaseRedemption(ctx context.Context, tx *gorm.DB, userID, discountCodeID string) error
	HasRedeemed(ctx context.Context, userID, discountCodeID string) (bool, error)
}

type campaignRepository struct {
	db *gorm.DB
}

func NewCampaignRepository(db *gorm.DB) CampaignRepository {
	return &campaignRepository{db: db}
}

func (r *campaignRepository) FindByID(ctx context.Context, id string) (*models.Campaign, error) {
	var found models.Campaign
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find campaign: %w", err)
	}
	return &found, nil
}

func (r *campaignRepository) List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.Campaign, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		if search != "" {
			pattern := "%" + strings.ToLower(search) + "%"
			tx = tx.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern)
		}
		if active != nil {
			tx = tx.Where("active = ?", *active)
		}
		return tx
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.Campaign{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count campaigns: %w", err)
	}

	var rows []models.Campaign
	if err := filter(r.db.WithContext(ctx)).
		Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}
	return rows, total, nil
}

func (r *campaignRepository) Create(ctx context.Context, tx *gorm.DB, c *models.Campaign) error {
	if err := tx.WithContext(ctx).Create(c).Error; err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}
	return nil
}

func (r *campaignRepository) Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).Model(&models.Campaign{}).
		Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}
	return nil
}

func (r *campaignRepository) Delete(ctx context.Context, tx *gorm.DB, id string) error {
	if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&models.Campaign{}).Error; err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}
	return nil
}

func (r *campaignRepository) ListActive(ctx context.Context, now time.Time, page, pageSize int) ([]models.Campaign, int64, error) {
	filter := func(tx *gorm.DB) *gorm.DB {
		return tx.Where("active = ? AND starts_at <= ? AND ends_at > ?", true, now, now)
	}

	var total int64
	if err := filter(r.db.WithContext(ctx).Model(&models.Campaign{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count active campaigns: %w", err)
	}

	var rows []models.Campaign
	if err := filter(r.db.WithContext(ctx)).
		Order("starts_at").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list active campaigns: %w", err)
	}
	return rows, total, nil
}

func (r *campaignRepository) FindActiveByID(ctx context.Context, id string, now time.Time) (*models.Campaign, error) {
	var found models.Campaign
	err := r.db.WithContext(ctx).
		Where("id = ? AND active = ? AND starts_at <= ? AND ends_at > ?", id, true, now, now).
		First(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find active campaign: %w", err)
	}
	return &found, nil
}

func (r *campaignRepository) AttachCombo(ctx context.Context, tx *gorm.DB, campaignID, comboID string, promoPrice *int64) error {
	link := models.CampaignCombo{CampaignID: campaignID, ComboID: comboID, PromoPrice: promoPrice}
	err := tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "combo_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"promo_price"}),
	}).Create(&link).Error
	if err != nil {
		return fmt.Errorf("attach campaign combo: %w", err)
	}
	return nil
}

func (r *campaignRepository) DetachCombo(ctx context.Context, tx *gorm.DB, campaignID, comboID string) error {
	if err := tx.WithContext(ctx).
		Where("campaign_id = ? AND combo_id = ?", campaignID, comboID).
		Delete(&models.CampaignCombo{}).Error; err != nil {
		return fmt.Errorf("detach campaign combo: %w", err)
	}
	return nil
}

func (r *campaignRepository) ListCombos(ctx context.Context, campaignID string) ([]models.CampaignCombo, error) {
	var rows []models.CampaignCombo
	if err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list campaign combos: %w", err)
	}
	return rows, nil
}

func (r *campaignRepository) AttachArticle(ctx context.Context, tx *gorm.DB, campaignID, articleID string) error {
	link := models.CampaignArticle{CampaignID: campaignID, ArticleID: articleID}
	err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error
	if err != nil {
		return fmt.Errorf("attach campaign article: %w", err)
	}
	return nil
}

func (r *campaignRepository) DetachArticle(ctx context.Context, tx *gorm.DB, campaignID, articleID string) error {
	if err := tx.WithContext(ctx).
		Where("campaign_id = ? AND article_id = ?", campaignID, articleID).
		Delete(&models.CampaignArticle{}).Error; err != nil {
		return fmt.Errorf("detach campaign article: %w", err)
	}
	return nil
}

func (r *campaignRepository) ListArticles(ctx context.Context, campaignID string) ([]models.CampaignArticle, error) {
	var rows []models.CampaignArticle
	if err := r.db.WithContext(ctx).Where("campaign_id = ?", campaignID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list campaign articles: %w", err)
	}
	return rows, nil
}

func (r *campaignRepository) ClaimRedemption(ctx context.Context, tx *gorm.DB, userID, discountCodeID string) error {
	redemption := models.DiscountRedemption{UserID: userID, DiscountCodeID: discountCodeID}
	if err := tx.WithContext(ctx).Create(&redemption).Error; err != nil {
		return fmt.Errorf("claim discount redemption: %w", err)
	}
	return nil
}

func (r *campaignRepository) ReleaseRedemption(ctx context.Context, tx *gorm.DB, userID, discountCodeID string) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND discount_code_id = ?", userID, discountCodeID).
		Delete(&models.DiscountRedemption{}).Error; err != nil {
		return fmt.Errorf("release discount redemption: %w", err)
	}
	return nil
}

func (r *campaignRepository) HasRedeemed(ctx context.Context, userID, discountCodeID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.DiscountRedemption{}).
		Where("user_id = ? AND discount_code_id = ?", userID, discountCodeID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check discount redemption: %w", err)
	}
	return count > 0, nil
}
