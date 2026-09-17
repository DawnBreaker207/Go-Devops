package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type ArticleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Create(tx *gorm.DB, a *models.Article) error {
	return tx.Create(a).Error
}

// List: publicOnly restricts to status=published for the public-facing endpoint.
func (r *ArticleRepository) List(ctx context.Context, publicOnly bool, articleType string, limit, offset int) ([]models.Article, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Article{})
	if publicOnly {
		q = q.Where("status = ?", models.ArticlePublished)
	}
	if articleType != "" {
		q = q.Where("type = ?", articleType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var articles []models.Article
	err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&articles).Error
	return articles, total, err
}

func (r *ArticleRepository) FindByID(ctx context.Context, id string) (*models.Article, error) {
	var a models.Article
	if err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ArticleRepository) FindBySlug(ctx context.Context, slug string) (*models.Article, error) {
	var a models.Article
	if err := r.db.WithContext(ctx).First(&a, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// SlugTaken checks across not-yet-deleted articles only, matching idx_articles_slug.
func (r *ArticleRepository) SlugTaken(ctx context.Context, slug, excludeID string) (bool, error) {
	q := r.db.WithContext(ctx).Model(&models.Article{}).Where("slug = ?", slug)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	err := q.Count(&count).Error
	return count > 0, err
}

func (r *ArticleRepository) Update(tx *gorm.DB, a *models.Article) error {
	return tx.Model(a).Select(
		"title", "slug", "summary", "thumbnail_url", "content", "type", "status", "updated_at",
	).Updates(a).Error
}

func (r *ArticleRepository) Delete(tx *gorm.DB, id string) error {
	return tx.Delete(&models.Article{}, "id = ?", id).Error
}

// IncrViews: fire-and-forget counter, no locking (ADVANCED_FEATURES_DISCUSSION.md Phần 1.4).
func (r *ArticleRepository) IncrViews(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Exec(`UPDATE articles SET views = views + 1 WHERE id = ?`, id).Error
}
