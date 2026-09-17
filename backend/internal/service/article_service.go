package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// Caching this endpoint (movie/showtime-style generation-key cache) is left
// for later: article reads are far lower-traffic than the public catalog,
// and Views is a live-incrementing counter that a cached response would
// immediately go stale on anyway.

type ArticleService interface {
	List(ctx context.Context, publicOnly bool, articleType string, query dto.PageQuery) ([]dto.ArticleResponse, int64, error)
	GetByID(ctx context.Context, id string) (*dto.ArticleResponse, error)
	// GetBySlug is the public read: it increments the view counter and only
	// ever returns a status=published article.
	GetBySlug(ctx context.Context, slug string) (*dto.ArticleResponse, error)
	Create(ctx context.Context, authorID string, req dto.ArticleRequest) (*dto.ArticleResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateArticleRequest) (*dto.ArticleResponse, error)
	Delete(ctx context.Context, id string) error
}

type articleService struct {
	db   *gorm.DB
	repo *repository.ArticleRepository
}

func NewArticleService(db *gorm.DB, repo *repository.ArticleRepository) ArticleService {
	return &articleService{db: db, repo: repo}
}

func (s *articleService) List(ctx context.Context, publicOnly bool, articleType string, query dto.PageQuery) ([]dto.ArticleResponse, int64, error) {
	rows, total, err := s.repo.List(ctx, publicOnly, articleType, query.PageSize, query.Offset())
	if err != nil {
		return nil, 0, err
	}
	return dto.NewArticleResponses(rows), total, nil
}

func (s *articleService) GetByID(ctx context.Context, id string) (*dto.ArticleResponse, error) {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, apperrors.ErrArticleNotFound
	}
	result := dto.NewArticleResponse(a)
	return &result, nil
}

func (s *articleService) GetBySlug(ctx context.Context, slug string) (*dto.ArticleResponse, error) {
	a, err := s.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if a == nil || a.Status != models.ArticlePublished {
		return nil, apperrors.ErrArticleNotFound
	}
	// Fire-and-forget: a view lost to a race is not worth serializing reads on.
	go func() { _ = s.repo.IncrViews(context.WithoutCancel(ctx), a.ID) }()
	result := dto.NewArticleResponse(a)
	return &result, nil
}

func (s *articleService) Create(ctx context.Context, authorID string, req dto.ArticleRequest) (*dto.ArticleResponse, error) {
	slug, err := s.resolveSlug(ctx, req.Slug, req.Title, "")
	if err != nil {
		return nil, err
	}
	articleType := req.Type
	if articleType == "" {
		articleType = models.ArticleTypeNews
	}
	status := req.Status
	if status == "" {
		status = models.ArticleDraft
	}
	a := &models.Article{
		Title: strings.TrimSpace(req.Title), Slug: slug, Summary: strings.TrimSpace(req.Summary),
		ThumbnailURL: strings.TrimSpace(req.ThumbnailURL), Content: req.Content, AuthorID: authorID,
		Type: articleType, Status: status,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, a); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = a.ID
			rec.After = map[string]any{"title": a.Title, "slug": a.Slug, "status": a.Status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewArticleResponse(a)
	return &result, nil
}

func (s *articleService) Update(ctx context.Context, id string, req dto.UpdateArticleRequest) (*dto.ArticleResponse, error) {
	var a *models.Article
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrArticleNotFound
		}
		before := map[string]any{"status": current.Status, "slug": current.Slug}

		if req.Title != nil {
			current.Title = strings.TrimSpace(*req.Title)
		}
		if req.Slug != nil {
			slug, err := s.resolveSlug(ctx, *req.Slug, current.Title, id)
			if err != nil {
				return err
			}
			current.Slug = slug
		}
		if req.Summary != nil {
			current.Summary = strings.TrimSpace(*req.Summary)
		}
		if req.ThumbnailURL != nil {
			current.ThumbnailURL = strings.TrimSpace(*req.ThumbnailURL)
		}
		if req.Content != nil {
			current.Content = *req.Content
		}
		if req.Type != nil {
			current.Type = *req.Type
		}
		if req.Status != nil {
			current.Status = *req.Status
		}
		if err := s.repo.Update(tx, current); err != nil {
			return err
		}
		a = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"status": a.Status, "slug": a.Slug}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewArticleResponse(a)
	return &result, nil
}

func (s *articleService) Delete(ctx context.Context, id string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrArticleNotFound
		}
		if err := s.repo.Delete(tx, id); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.After = map[string]any{"deleted": true}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugInvalidChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// resolveSlug: an explicit slug is slugified and used as is (still checked
// for a collision); an empty one is derived from the title, with a numeric
// suffix appended on a collision (Phần 1.4).
func (s *articleService) resolveSlug(ctx context.Context, explicit, title, excludeID string) (string, error) {
	base := slugify(explicit)
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		return "", apperrors.Validation("could not derive a slug from title")
	}
	slug := base
	for i := 2; ; i++ {
		taken, err := s.repo.SlugTaken(ctx, slug, excludeID)
		if err != nil {
			return "", err
		}
		if !taken {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
