package service

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// ComboService sells concession/combo products independently of the ticket
// booking flow (F5): a combo order failing must never roll back or block a
// booking, and a booking failing must never roll back a combo order. Neither
// side opens a shared transaction with the other.
type ComboService interface {
	ListActive(ctx context.Context) ([]dto.ComboResponse, error)
	CreateOrder(ctx context.Context, userID string, req dto.CreateComboOrderRequest) (*dto.ComboOrderResponse, error)
	ListMyOrders(ctx context.Context, userID string, q dto.PageQuery) ([]dto.ComboOrderResponse, int64, error)

	// Operator catalogue management (admin AND staff, like halls and showtimes).
	AdminList(ctx context.Context, q dto.AdminComboListQuery) ([]dto.ComboResponse, int64, error)
	AdminGet(ctx context.Context, id string) (*dto.ComboResponse, error)
	AdminCreate(ctx context.Context, req dto.CreateComboRequest) (*dto.ComboResponse, error)
	AdminUpdate(ctx context.Context, id string, req dto.UpdateComboRequest) (*dto.ComboResponse, error)
	AdminDelete(ctx context.Context, id string) error
}

type comboService struct {
	// db is only for the operator catalogue writes below, which pair a change with
	// its audit row. The customer-facing order path deliberately keeps its own
	// single-statement transaction inside ComboOrderRepository.
	db          *gorm.DB
	combos      repository.ComboRepository
	orders      repository.ComboOrderRepository
	bookingRepo repository.BookingRepository // optional ownership check only; may be nil
}

// NewComboService: bookingRepo may be nil, which simply skips the ownership
// check when an order names a booking_id.
func NewComboService(db *gorm.DB, combos repository.ComboRepository, orders repository.ComboOrderRepository, bookingRepo repository.BookingRepository) ComboService {
	return &comboService{db: db, combos: combos, orders: orders, bookingRepo: bookingRepo}
}

func (s *comboService) ListActive(ctx context.Context) ([]dto.ComboResponse, error) {
	combos, err := s.combos.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return dto.NewComboResponses(combos), nil
}

func (s *comboService) CreateOrder(ctx context.Context, userID string, req dto.CreateComboOrderRequest) (*dto.ComboOrderResponse, error) {
	if req.BookingID != "" && s.bookingRepo != nil {
		booking, err := s.bookingRepo.FindByIDAndUser(ctx, req.BookingID, userID)
		if err != nil {
			return nil, err
		}
		if booking == nil {
			return nil, apperrors.ErrBookingNotFound
		}
	}

	ids := make([]string, 0, len(req.Items))
	qtyByCombo := make(map[string]int, len(req.Items))
	for _, item := range req.Items {
		if _, ok := qtyByCombo[item.ComboID]; !ok {
			ids = append(ids, item.ComboID)
		}
		qtyByCombo[item.ComboID] += item.Quantity
	}

	found, err := s.combos.FindActiveByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(found) != len(ids) {
		return nil, apperrors.ErrComboInactive
	}

	var total int64
	items := make([]models.ComboOrderItem, 0, len(ids))
	for _, id := range ids {
		combo := found[id]
		qty := qtyByCombo[id]
		total += combo.Price * int64(qty)
		items = append(items, models.ComboOrderItem{
			ComboID:   combo.ID,
			ComboName: combo.Name,
			Quantity:  qty,
			UnitPrice: combo.Price,
		})
	}
	if len(items) == 0 {
		return nil, apperrors.ErrComboOrderEmpty
	}

	order := &models.ComboOrder{
		UserID: userID,
		Status: models.ComboOrderConfirmed,
		Total:  total,
	}
	if req.BookingID != "" {
		order.BookingID = &req.BookingID
	}
	if err := s.orders.Create(ctx, order, items); err != nil {
		return nil, err
	}

	result := dto.NewComboOrderResponse(order, items)
	return &result, nil
}

func (s *comboService) ListMyOrders(ctx context.Context, userID string, q dto.PageQuery) ([]dto.ComboOrderResponse, int64, error) {
	orders, total, err := s.orders.ListByUser(ctx, userID, q.Page, q.PageSize)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(orders))
	for i := range orders {
		ids = append(ids, orders[i].ID)
	}
	itemsByOrder, err := s.orders.ItemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.ComboOrderResponse, 0, len(orders))
	for i := range orders {
		result = append(result, dto.NewComboOrderResponse(&orders[i], itemsByOrder[orders[i].ID]))
	}
	return result, total, nil
}

/* -------------------------------------------------------------------------- */
/* Operator catalogue management                                              */
/* -------------------------------------------------------------------------- */

// AdminList shows the catalogue an operator manages: paged, searchable, and it
// includes INACTIVE products, which the public ListActive never returns.
func (s *comboService) AdminList(ctx context.Context, q dto.AdminComboListQuery) ([]dto.ComboResponse, int64, error) {
	combos, total, err := s.combos.List(ctx, q.Page, q.PageSize, strings.TrimSpace(q.Search), q.Active)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewComboResponses(combos), total, nil
}

func (s *comboService) AdminGet(ctx context.Context, id string) (*dto.ComboResponse, error) {
	combo, err := s.combos.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if combo == nil {
		return nil, apperrors.ErrComboNotFound
	}
	result := dto.NewComboResponse(combo)
	return &result, nil
}

func (s *comboService) AdminCreate(ctx context.Context, req dto.CreateComboRequest) (*dto.ComboResponse, error) {
	combo := &models.Combo{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Price:       req.Price,
		ImageURL:    strings.TrimSpace(req.ImageURL),
		// A product added from the admin screen goes on sale unless told otherwise.
		Active: req.Active == nil || *req.Active,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.combos.Create(ctx, tx, combo); err != nil {
			return err
		}
		return s.auditCombo(ctx, tx, combo.ID, nil, comboAuditFields(combo))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewComboResponse(combo)
	return &result, nil
}

func (s *comboService) AdminUpdate(ctx context.Context, id string, req dto.UpdateComboRequest) (*dto.ComboResponse, error) {
	if req.IsEmpty() {
		return nil, apperrors.Validation("nothing to update")
	}

	current, err := s.combos.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, apperrors.ErrComboNotFound
	}
	before := comboAuditFields(current)

	// Only the fields the caller actually sent; a map (not a struct) so price 0
	// and active false are written rather than skipped as Go zero values.
	fields := make(map[string]any, 5)
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
		fields["name"] = current.Name
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
		fields["description"] = current.Description
	}
	if req.Price != nil {
		current.Price = *req.Price
		fields["price"] = current.Price
	}
	if req.ImageURL != nil {
		current.ImageURL = strings.TrimSpace(*req.ImageURL)
		fields["image_url"] = current.ImageURL
	}
	if req.Active != nil {
		current.Active = *req.Active
		fields["active"] = current.Active
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.combos.Update(ctx, tx, id, fields); err != nil {
			return err
		}
		return s.auditCombo(ctx, tx, id, before, comboAuditFields(current))
	})
	if err != nil {
		return nil, err
	}

	result := dto.NewComboResponse(current)
	return &result, nil
}

// AdminDelete soft-deletes: combo_order_items still point here by FK, and a past
// receipt must stay readable after a product is retired.
func (s *comboService) AdminDelete(ctx context.Context, id string) error {
	current, err := s.combos.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if current == nil {
		return apperrors.ErrComboNotFound
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.combos.SoftDelete(ctx, tx, id); err != nil {
			return err
		}
		return s.auditCombo(ctx, tx, id, comboAuditFields(current), nil)
	})
}

// comboAuditFields is the audited projection of a product. Deliberately a plain
// map, never the model: audit.Record.Before/After are map[string]any and a model
// would leak whatever the struct grows later.
func comboAuditFields(c *models.Combo) map[string]any {
	return map[string]any{
		"name":   c.Name,
		"price":  c.Price,
		"active": c.Active,
	}
}

// auditCombo is a no-op when there is no middleware.Audit in play (e.g. a direct
// service-level test), matching how the rest of the services treat a miss.
func (s *comboService) auditCombo(ctx context.Context, tx *gorm.DB, id string, before, after map[string]any) error {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return nil
	}
	rec.ResourceID = id
	rec.Before = before
	rec.After = after
	return audit.In(ctx, tx, rec)
}
