package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// CampaignService owns campaign catalogue (admin CRUD, attach/detach) and public read.
// PromoPrice is display only; discount side effects live in DiscountService.
type CampaignService interface {
	AdminList(ctx context.Context, q dto.CampaignListQuery) ([]dto.CampaignResponse, int64, error)
	AdminGet(ctx context.Context, id string) (*dto.CampaignDetailResponse, error)
	AdminCreate(ctx context.Context, req dto.CreateCampaignRequest) (*dto.CampaignResponse, error)
	AdminUpdate(ctx context.Context, id string, req dto.UpdateCampaignRequest) (*dto.CampaignResponse, error)
	AdminDelete(ctx context.Context, id string) error

	AttachCombo(ctx context.Context, campaignID, comboID string, req dto.AttachComboRequest) (*dto.CampaignComboResponse, error)
	DetachCombo(ctx context.Context, campaignID, comboID string) error
	AttachArticle(ctx context.Context, campaignID, articleID string) (*dto.ArticleResponse, error)
	DetachArticle(ctx context.Context, campaignID, articleID string) error
	AttachDiscountCode(ctx context.Context, campaignID, discountCodeID string) (*dto.DiscountCodeResponse, error)
	DetachDiscountCode(ctx context.Context, campaignID, discountCodeID string) error

	PublicList(ctx context.Context, q dto.PageQuery) ([]dto.CampaignPublicResponse, int64, error)
	PublicGet(ctx context.Context, id string) (*dto.CampaignPublicResponse, error)
}

type campaignService struct {
	db        *gorm.DB
	campaigns repository.CampaignRepository
	discounts repository.DiscountRepository
	combos    repository.ComboRepository
	articles  *repository.ArticleRepository
	// now is injectable so the window tests do not depend on wall clock.
	now func() time.Time
}

func NewCampaignService(db *gorm.DB, campaigns repository.CampaignRepository, discounts repository.DiscountRepository,
	combos repository.ComboRepository, articles *repository.ArticleRepository) CampaignService {
	return &campaignService{db: db, campaigns: campaigns, discounts: discounts, combos: combos, articles: articles, now: time.Now}
}

func (s *campaignService) AdminList(ctx context.Context, q dto.CampaignListQuery) ([]dto.CampaignResponse, int64, error) {
	rows, total, err := s.campaigns.List(ctx, q.Page, q.PageSize, strings.TrimSpace(q.Search), q.Active)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewCampaignResponses(rows), total, nil
}

func (s *campaignService) AdminGet(ctx context.Context, id string) (*dto.CampaignDetailResponse, error) {
	found, err := s.campaigns.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, apperrors.ErrCampaignNotFound
	}
	return s.detail(ctx, found, false)
}

func (s *campaignService) AdminCreate(ctx context.Context, req dto.CreateCampaignRequest) (*dto.CampaignResponse, error) {
	if !req.EndsAt.After(req.StartsAt) {
		return nil, apperrors.Validation("ends_at must be after starts_at")
	}
	perUserLimit := 1
	if req.PerUserLimit != nil {
		perUserLimit = *req.PerUserLimit
	}
	created := &models.Campaign{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		StartsAt:     req.StartsAt,
		EndsAt:       req.EndsAt,
		PerUserLimit: perUserLimit,
		Active:       req.Active != nil && *req.Active,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.Create(ctx, tx, created); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.create_campaign", created.ID, nil, campaignAuditFields(created))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewCampaignResponse(created)
	return &result, nil
}

func (s *campaignService) AdminUpdate(ctx context.Context, id string, req dto.UpdateCampaignRequest) (*dto.CampaignResponse, error) {
	if req.IsEmpty() {
		return nil, apperrors.Validation("nothing to update")
	}

	current, err := s.campaigns.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, apperrors.ErrCampaignNotFound
	}
	before := campaignAuditFields(current)

	fields := make(map[string]any, 6)
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
		fields["name"] = current.Name
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
		fields["description"] = current.Description
	}
	if req.StartsAt != nil {
		current.StartsAt = *req.StartsAt
		fields["starts_at"] = current.StartsAt
	}
	if req.EndsAt != nil {
		current.EndsAt = *req.EndsAt
		fields["ends_at"] = current.EndsAt
	}
	if req.PerUserLimit != nil {
		current.PerUserLimit = *req.PerUserLimit
		fields["per_user_limit"] = current.PerUserLimit
	}
	if req.Active != nil {
		current.Active = *req.Active
		fields["active"] = current.Active
	}

	// Re-validate result window: changing one side can break it.
	if !current.EndsAt.After(current.StartsAt) {
		return nil, apperrors.Validation("ends_at must be after starts_at")
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.Update(ctx, tx, id, fields); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.update_campaign", id, before, campaignAuditFields(current))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewCampaignResponse(current)
	return &result, nil
}

// AdminDelete hard-deletes; linked codes become standalone (campaign_id NULL).
func (s *campaignService) AdminDelete(ctx context.Context, id string) error {
	current, err := s.campaigns.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return apperrors.ErrCampaignNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.Delete(ctx, tx, id); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.delete_campaign", id, campaignAuditFields(current), nil)
	})
}

func (s *campaignService) AttachCombo(ctx context.Context, campaignID, comboID string, req dto.AttachComboRequest) (*dto.CampaignComboResponse, error) {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	combo, err := s.combos.FindByID(ctx, comboID)
	if err != nil {
		return nil, err
	}
	if combo == nil {
		return nil, apperrors.ErrComboNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.AttachCombo(ctx, tx, campaignID, comboID, req.PromoPrice); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.attach_campaign_combo", campaignID, nil,
			map[string]any{"combo_id": comboID, "promo_price": req.PromoPrice})
	})
	if err != nil {
		return nil, err
	}

	result := dto.CampaignComboResponse{ComboID: combo.ID, Name: combo.Name, Price: combo.Price, PromoPrice: req.PromoPrice}
	return &result, nil
}

func (s *campaignService) DetachCombo(ctx context.Context, campaignID, comboID string) error {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.DetachCombo(ctx, tx, campaignID, comboID); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.detach_campaign_combo", campaignID, map[string]any{"combo_id": comboID}, nil)
	})
}

func (s *campaignService) AttachArticle(ctx context.Context, campaignID, articleID string) (*dto.ArticleResponse, error) {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	article, err := s.articles.FindByID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, apperrors.ErrArticleNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.AttachArticle(ctx, tx, campaignID, articleID); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.attach_campaign_article", campaignID, nil, map[string]any{"article_id": articleID})
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewArticleResponse(article)
	return &result, nil
}

func (s *campaignService) DetachArticle(ctx context.Context, campaignID, articleID string) error {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.campaigns.DetachArticle(ctx, tx, campaignID, articleID); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.detach_campaign_article", campaignID, map[string]any{"article_id": articleID}, nil)
	})
}

func (s *campaignService) AttachDiscountCode(ctx context.Context, campaignID, discountCodeID string) (*dto.DiscountCodeResponse, error) {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	code, err := s.discounts.FindByID(ctx, discountCodeID)
	if err != nil {
		return nil, err
	}
	if code == nil {
		return nil, apperrors.ErrDiscountNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.discounts.Update(ctx, tx, discountCodeID, map[string]any{"campaign_id": campaignID}); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.attach_campaign_discount", campaignID, nil, map[string]any{"discount_code_id": discountCodeID})
	})
	if err != nil {
		return nil, err
	}

	code.CampaignID = &campaignID
	result := dto.NewDiscountCodeResponse(code)
	return &result, nil
}

func (s *campaignService) DetachDiscountCode(ctx context.Context, campaignID, discountCodeID string) error {
	if _, err := s.mustCampaign(ctx, campaignID); err != nil {
		return err
	}
	code, err := s.discounts.FindByID(ctx, discountCodeID)
	if err != nil {
		return err
	}
	if code == nil {
		return apperrors.ErrDiscountNotFound
	}
	if code.CampaignID == nil || *code.CampaignID != campaignID {
		return apperrors.NotFound("this discount code is not attached to this campaign")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.discounts.Update(ctx, tx, discountCodeID, map[string]any{"campaign_id": nil}); err != nil {
			return err
		}
		return s.auditCampaign(ctx, tx, "admin.detach_campaign_discount", campaignID, map[string]any{"discount_code_id": discountCodeID}, nil)
	})
}

func (s *campaignService) mustCampaign(ctx context.Context, id string) (*models.Campaign, error) {
	found, err := s.campaigns.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, apperrors.ErrCampaignNotFound
	}
	return found, nil
}

func (s *campaignService) PublicList(ctx context.Context, q dto.PageQuery) ([]dto.CampaignPublicResponse, int64, error) {
	now := s.now()
	rows, total, err := s.campaigns.ListActive(ctx, now, q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.CampaignPublicResponse, 0, len(rows))
	for i := range rows {
		resp, err := s.publicDetail(ctx, &rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *resp)
	}
	return out, total, nil
}

func (s *campaignService) PublicGet(ctx context.Context, id string) (*dto.CampaignPublicResponse, error) {
	found, err := s.campaigns.FindActiveByID(ctx, id, s.now())
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, apperrors.ErrCampaignNotFound
	}
	return s.publicDetail(ctx, found)
}

// detail: admin view, no visibility filtering.
func (s *campaignService) detail(ctx context.Context, c *models.Campaign, _ bool) (*dto.CampaignDetailResponse, error) {
	codes, err := s.discounts.ListByCampaignID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	combos, err := s.comboResponses(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	articleRows, err := s.campaigns.ListArticles(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	articles := make([]dto.ArticleResponse, 0, len(articleRows))
	for _, row := range articleRows {
		a, err := s.articles.FindByID(ctx, row.ArticleID)
		if err != nil {
			return nil, err
		}
		if a == nil {
			continue
		}
		articles = append(articles, dto.NewArticleResponse(a))
	}

	return &dto.CampaignDetailResponse{
		CampaignResponse: dto.NewCampaignResponse(c),
		DiscountCodes:    dto.NewDiscountCodeResponses(codes),
		Combos:           combos,
		Articles:         articles,
	}, nil
}

// publicDetail: only published articles and active codes.
func (s *campaignService) publicDetail(ctx context.Context, c *models.Campaign) (*dto.CampaignPublicResponse, error) {
	codes, err := s.discounts.ListByCampaignID(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	publicCodes := make([]dto.CampaignPublicDiscountCode, 0, len(codes))
	for _, code := range codes {
		if !code.Active {
			continue
		}
		entry := dto.CampaignPublicDiscountCode{Code: code.Code}
		if code.MaxUses != nil {
			remaining := *code.MaxUses - code.UsedCount
			if remaining < 0 {
				remaining = 0
			}
			entry.Remaining = &remaining
		}
		publicCodes = append(publicCodes, entry)
	}

	combos, err := s.comboResponses(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	articleRows, err := s.campaigns.ListArticles(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	articles := make([]dto.ArticleResponse, 0, len(articleRows))
	for _, row := range articleRows {
		a, err := s.articles.FindByID(ctx, row.ArticleID)
		if err != nil {
			return nil, err
		}
		// Only published articles are visible to customers.
		if a == nil || a.Status != models.ArticlePublished {
			continue
		}
		articles = append(articles, dto.NewArticleResponse(a))
	}

	return &dto.CampaignPublicResponse{
		ID:            c.ID,
		Name:          c.Name,
		Description:   c.Description,
		StartsAt:      c.StartsAt,
		EndsAt:        c.EndsAt,
		DiscountCodes: publicCodes,
		Combos:        combos,
		Articles:      articles,
	}, nil
}

func (s *campaignService) comboResponses(ctx context.Context, campaignID string) ([]dto.CampaignComboResponse, error) {
	links, err := s.campaigns.ListCombos(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CampaignComboResponse, 0, len(links))
	for _, link := range links {
		combo, err := s.combos.FindByID(ctx, link.ComboID)
		if err != nil {
			return nil, err
		}
		if combo == nil {
			continue
		}
		out = append(out, dto.CampaignComboResponse{
			ComboID: combo.ID, Name: combo.Name, Price: combo.Price, PromoPrice: link.PromoPrice,
		})
	}
	return out, nil
}

func campaignAuditFields(c *models.Campaign) map[string]any {
	return map[string]any{
		"name": c.Name, "starts_at": c.StartsAt, "ends_at": c.EndsAt,
		"active": c.Active, "per_user_limit": c.PerUserLimit,
	}
}

func (s *campaignService) auditCampaign(ctx context.Context, tx *gorm.DB, action, id string, before, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return nil
	}
	rec.Action = action
	rec.ResourceID = id
	rec.Before = before
	rec.After = after
	return audit.In(ctx, tx, rec)
}
