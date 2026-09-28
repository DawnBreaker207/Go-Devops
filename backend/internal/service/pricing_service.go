package service

import (
	"context"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// PricingService: global base price per seat type + adjustment rules + preview.
type PricingService interface {
	AdminGetBasePrices(ctx context.Context) ([]dto.BasePriceResponse, error)
	AdminSetBasePrices(ctx context.Context, req dto.BasePriceRequest) ([]dto.BasePriceResponse, error)

	// PublicPrices: customer price page, global bases + cheapest configured.
	PublicPrices(ctx context.Context) (*dto.GlobalPriceListResponse, error)

	AdminListRules(ctx context.Context, q dto.PricingRuleListQuery) ([]dto.PricingRuleResponse, int64, error)
	AdminGetRule(ctx context.Context, id string) (*dto.PricingRuleResponse, error)
	AdminCreateRule(ctx context.Context, req dto.CreatePricingRuleRequest) (*dto.PricingRuleResponse, error)
	AdminUpdateRule(ctx context.Context, id string, req dto.UpdatePricingRuleRequest) (*dto.PricingRuleResponse, error)
	AdminDeleteRule(ctx context.Context, id string) error

	// Quote previews base + matching rules (priority first) + floored final. Read-only.
	Quote(ctx context.Context, showtimeID, seatType string) (*dto.PricingQuoteResponse, error)

	// QuotePrices is Quote's bulk sibling for hold/counter/seatmap; shares priceFor.
	QuotePrices(ctx context.Context, showtime *models.Showtime, seatTypes []string) (map[string]SeatTypePrice, error)
}

// SeatTypePrice: final price; Configured false means no positive base price yet.
type SeatTypePrice struct {
	Final      int64
	Configured bool
}

type pricingService struct {
	db       *gorm.DB
	pricing  repository.PricingRepository
	showtime *repository.ShowtimeRepository
	// location is cinema-local timezone for rule matching.
	location *time.Location
	// cache is nil when Redis is not configured; bumpCatalog tolerates that.
	cache *cache.Cache
}

func NewPricingService(db *gorm.DB, pricing repository.PricingRepository, showtime *repository.ShowtimeRepository,
	location *time.Location, c *cache.Cache) PricingService {
	return &pricingService{db: db, pricing: pricing, showtime: showtime, location: location, cache: c}
}

func (s *pricingService) AdminGetBasePrices(ctx context.Context) ([]dto.BasePriceResponse, error) {
	prices, err := s.pricing.GetBasePrices(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewBasePriceResponses(prices), nil
}

func (s *pricingService) AdminSetBasePrices(ctx context.Context, req dto.BasePriceRequest) ([]dto.BasePriceResponse, error) {
	if len(req.Prices) == 0 {
		return nil, apperrors.Validation("nothing to update")
	}
	for seatType, price := range req.Prices {
		if !slices.Contains(models.AllSeatTypes, seatType) {
			return nil, apperrors.ErrSeatTypeInvalid
		}
		if price < 0 {
			return nil, apperrors.Validation("price can not be negative")
		}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for seatType, price := range req.Prices {
			if err := s.pricing.UpsertBasePrice(ctx, tx, seatType, price); err != nil {
				return err
			}
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.Action = "admin.set_base_price"
			rec.ResourceType = "seat_base_price"
			rec.After = map[string]any{"prices": req.Prices}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Base-price write is customer-visible; bump after commit like other price writes.
	bumpCatalog(ctx, s.cache)

	prices, err := s.pricing.GetBasePrices(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewBasePriceResponses(prices), nil
}

func (s *pricingService) PublicPrices(ctx context.Context) (*dto.GlobalPriceListResponse, error) {
	prices, err := s.pricing.GetBasePrices(ctx)
	if err != nil {
		return nil, err
	}
	result := dto.GlobalPriceListResponse{Prices: make(map[string]int64, len(prices))}
	for _, p := range prices {
		result.Prices[p.SeatType] = p.Price
		if p.Price > 0 && (result.FromPrice == 0 || p.Price < result.FromPrice) {
			result.FromPrice = p.Price
		}
	}
	return &result, nil
}

func (s *pricingService) AdminListRules(ctx context.Context, q dto.PricingRuleListQuery) ([]dto.PricingRuleResponse, int64, error) {
	q.Search = strings.ToLower(strings.TrimSpace(q.Search))
	rules, total, err := s.pricing.ListRules(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewPricingRuleResponses(rules), total, nil
}

func (s *pricingService) AdminGetRule(ctx context.Context, id string) (*dto.PricingRuleResponse, error) {
	found, err := s.pricing.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, apperrors.ErrPricingRuleNotFound
	}
	result := dto.NewPricingRuleResponse(found)
	return &result, nil
}

func (s *pricingService) AdminCreateRule(ctx context.Context, req dto.CreatePricingRuleRequest) (*dto.PricingRuleResponse, error) {
	startTime, err := parseClockTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := parseClockTime(req.EndTime)
	if err != nil {
		return nil, err
	}
	specificDate, err := parseSpecificDate(req.SpecificDate)
	if err != nil {
		return nil, err
	}
	if err := validateRuleTimeWindow(startTime, endTime); err != nil {
		return nil, err
	}

	var dayOfWeek *int16
	if req.DayOfWeek != nil {
		v := int16(*req.DayOfWeek)
		dayOfWeek = &v
	}

	created := &models.PricingRule{
		Name:         strings.TrimSpace(req.Name),
		DayOfWeek:    dayOfWeek,
		StartTime:    startTime,
		EndTime:      endTime,
		SpecificDate: specificDate,
		AdjustKind:   req.AdjustKind,
		AdjustValue:  req.AdjustValue,
		Priority:     req.Priority,
		Active:       req.Active == nil || *req.Active,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.pricing.CreateRule(ctx, tx, created); err != nil {
			return err
		}
		return s.auditRule(ctx, tx, "admin.create_pricing_rule", created.ID, nil, pricingRuleAuditFields(created))
	})
	if err != nil {
		return nil, err
	}
	bumpCatalog(ctx, s.cache)

	result := dto.NewPricingRuleResponse(created)
	return &result, nil
}

func (s *pricingService) AdminUpdateRule(ctx context.Context, id string, req dto.UpdatePricingRuleRequest) (*dto.PricingRuleResponse, error) {
	if req.IsEmpty() {
		return nil, apperrors.Validation("nothing to update")
	}

	current, err := s.pricing.GetRule(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, apperrors.ErrPricingRuleNotFound
	}
	before := pricingRuleAuditFields(current)

	fields := make(map[string]any, 8)
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
		fields["name"] = current.Name
	}
	if req.DayOfWeek != nil {
		v := int16(*req.DayOfWeek)
		current.DayOfWeek = &v
		fields["day_of_week"] = current.DayOfWeek
	}
	// Empty clears to NULL; nil leaves alone.
	if req.StartTime != nil {
		t, err := parseClockTime(req.StartTime)
		if err != nil {
			return nil, err
		}
		current.StartTime = t
		fields["start_time"] = t
	}
	if req.EndTime != nil {
		t, err := parseClockTime(req.EndTime)
		if err != nil {
			return nil, err
		}
		current.EndTime = t
		fields["end_time"] = t
	}
	if req.SpecificDate != nil {
		d, err := parseSpecificDate(req.SpecificDate)
		if err != nil {
			return nil, err
		}
		current.SpecificDate = d
		fields["specific_date"] = d
	}
	if req.AdjustKind != nil {
		current.AdjustKind = *req.AdjustKind
		fields["adjust_kind"] = current.AdjustKind
	}
	if req.AdjustValue != nil {
		current.AdjustValue = *req.AdjustValue
		fields["adjust_value"] = current.AdjustValue
	}
	if req.Priority != nil {
		current.Priority = *req.Priority
		fields["priority"] = current.Priority
	}
	if req.Active != nil {
		current.Active = *req.Active
		fields["active"] = current.Active
	}

	// Re-validate result: changing only end_time can break window.
	if err := validateRuleTimeWindow(current.StartTime, current.EndTime); err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.pricing.UpdateRule(ctx, tx, id, fields); err != nil {
			return err
		}
		return s.auditRule(ctx, tx, "admin.update_pricing_rule", id, before, pricingRuleAuditFields(current))
	})
	if err != nil {
		return nil, err
	}
	bumpCatalog(ctx, s.cache)

	result := dto.NewPricingRuleResponse(current)
	return &result, nil
}

func (s *pricingService) AdminDeleteRule(ctx context.Context, id string) error {
	current, err := s.pricing.GetRule(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return apperrors.ErrPricingRuleNotFound
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.pricing.DeleteRule(ctx, tx, id); err != nil {
			return err
		}
		return s.auditRule(ctx, tx, "admin.delete_pricing_rule", id, pricingRuleAuditFields(current), nil)
	})
	if err != nil {
		return err
	}
	bumpCatalog(ctx, s.cache)
	return nil
}

func pricingRuleAuditFields(r *models.PricingRule) map[string]any {
	return map[string]any{
		"name": r.Name, "adjust_kind": r.AdjustKind, "adjust_value": r.AdjustValue,
		"priority": r.Priority, "active": r.Active,
	}
}

func (s *pricingService) auditRule(ctx context.Context, tx *gorm.DB, action, id string, before, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return nil
	}
	rec.Action = action
	rec.ResourceType = "pricing_rule"
	rec.ResourceID = id
	rec.Before = before
	rec.After = after
	return audit.In(ctx, tx, rec)
}

func (s *pricingService) Quote(ctx context.Context, showtimeID, seatType string) (*dto.PricingQuoteResponse, error) {
	if !slices.Contains(models.AllSeatTypes, seatType) {
		return nil, apperrors.ErrSeatTypeInvalid
	}

	row, err := s.showtime.FindByID(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apperrors.ErrShowtimeNotFound
	}

	prices, err := s.pricing.GetBasePrices(ctx)
	if err != nil {
		return nil, err
	}
	var base int64
	for _, p := range prices {
		if p.SeatType == seatType {
			base = p.Price
			break
		}
	}

	rules, err := s.pricing.ListActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	loc := s.location
	if loc == nil {
		loc = time.UTC
	}
	startLocal := row.StartAt.In(loc)
	weekday := int16(startLocal.Weekday())
	dateStr := startLocal.Format("2006-01-02")
	clock := models.ClockTime(startLocal.Format("15:04:05"))

	final, applied := priceFor(base, rules, weekday, dateStr, clock)
	return &dto.PricingQuoteResponse{Base: base, Applied: applied, Final: final}, nil
}

// QuotePrices: bulk sibling; fetches bases+rules once, reuses priceFor.
func (s *pricingService) QuotePrices(ctx context.Context, showtime *models.Showtime, seatTypes []string) (map[string]SeatTypePrice, error) {
	prices, err := s.pricing.GetBasePrices(ctx)
	if err != nil {
		return nil, err
	}
	baseByType := make(map[string]int64, len(prices))
	for _, p := range prices {
		baseByType[p.SeatType] = p.Price
	}

	rules, err := s.pricing.ListActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	loc := s.location
	if loc == nil {
		loc = time.UTC
	}
	startLocal := showtime.StartAt.In(loc)
	weekday := int16(startLocal.Weekday())
	dateStr := startLocal.Format("2006-01-02")
	clock := models.ClockTime(startLocal.Format("15:04:05"))

	out := make(map[string]SeatTypePrice, len(seatTypes))
	for _, seatType := range seatTypes {
		base := baseByType[seatType]
		final, _ := priceFor(base, rules, weekday, dateStr, clock)
		out[seatType] = SeatTypePrice{Final: final, Configured: base > 0}
	}
	return out, nil
}

// priceFor: single base+rules to final price; rules ordered priority DESC, created_at ASC.
func priceFor(base int64, rules []models.PricingRule, weekday int16, dateStr string, clock models.ClockTime) (int64, []dto.AppliedPricingRule) {
	applied := make([]dto.AppliedPricingRule, 0, len(rules))
	running := base
	for _, rule := range rules {
		if !ruleMatches(&rule, weekday, dateStr, clock) {
			continue
		}
		switch rule.AdjustKind {
		case models.AdjustPercent:
			// Percent compounds on running total.
			running += running * rule.AdjustValue / 100
		case models.AdjustFixed:
			running += rule.AdjustValue
		default:
			continue
		}
		applied = append(applied, dto.AppliedPricingRule{
			RuleID: rule.ID, Name: rule.Name, AdjustKind: rule.AdjustKind, AdjustValue: rule.AdjustValue,
		})
	}
	if running < 0 {
		running = 0
	}
	return running, applied
}

// ruleMatches: specific_date wins over day_of_week; both nil matches every day.
func ruleMatches(rule *models.PricingRule, weekday int16, dateStr string, clock models.ClockTime) bool {
	dayMatches := true
	switch {
	case rule.SpecificDate != nil:
		dayMatches = rule.SpecificDate.Format("2006-01-02") == dateStr
	case rule.DayOfWeek != nil:
		dayMatches = *rule.DayOfWeek == weekday
	}
	if !dayMatches {
		return false
	}

	switch {
	case rule.StartTime != nil && rule.EndTime != nil:
		return clock >= *rule.StartTime && clock <= *rule.EndTime
	case rule.StartTime != nil:
		return clock >= *rule.StartTime
	case rule.EndTime != nil:
		return clock <= *rule.EndTime
	default:
		return true
	}
}

// parseClockTime accepts "HH:MM" or "HH:MM:SS"; nil/empty clears.
func parseClockTime(raw *string) (*models.ClockTime, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, trimmed); err == nil {
			ct := models.ClockTime(t.Format("15:04:05"))
			return &ct, nil
		}
	}
	return nil, apperrors.ErrPricingTimeInvalid
}

func parseSpecificDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, apperrors.ErrPricingDateInvalid
	}
	return &t, nil
}

func validateRuleTimeWindow(start, end *models.ClockTime) error {
	if start != nil && end != nil && *end <= *start {
		return apperrors.Validation("end_time must be after start_time")
	}
	return nil
}
