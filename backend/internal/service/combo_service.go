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
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// ComboService sells combos independently of ticket booking; neither side shares a transaction.
type ComboService interface {
	ListActive(ctx context.Context) ([]dto.ComboResponse, error)
	CreateOrder(ctx context.Context, userID string, req dto.CreateComboOrderRequest) (*dto.ComboOrderResponse, error)
	CounterSell(ctx context.Context, req dto.CounterComboOrderRequest) (*dto.ComboOrderResponse, error)
	PendingPickups(ctx context.Context, date, search string) ([]dto.ComboPickupResponse, error)
	CollectOrder(ctx context.Context, id string) (*dto.ComboOrderResponse, error)
	ListMyOrders(ctx context.Context, userID string, q dto.PageQuery) ([]dto.ComboOrderResponse, int64, error)

	// Operator catalogue management (admin AND staff, like halls and showtimes).
	AdminList(ctx context.Context, q dto.AdminComboListQuery) ([]dto.ComboResponse, int64, error)
	AdminGet(ctx context.Context, id string) (*dto.ComboResponse, error)
	AdminCreate(ctx context.Context, req dto.CreateComboRequest) (*dto.ComboResponse, error)
	AdminUpdate(ctx context.Context, id string, req dto.UpdateComboRequest) (*dto.ComboResponse, error)
	AdminDelete(ctx context.Context, id string) error
}

type comboService struct {
	// db is only for operator writes (change + audit row in one tx).
	db          *gorm.DB
	combos      repository.ComboRepository
	orders      repository.ComboOrderRepository
	bookingRepo repository.BookingRepository // optional ownership check only; may be nil
}

// NewComboService: nil bookingRepo skips ownership check.
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

	items, total, err := s.buildOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	order := &models.ComboOrder{
		UserID:      &userID,
		Status:      models.ComboOrderConfirmed,
		Total:       total,
		SoldChannel: models.SoldChannelOnline,
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

// CounterSell: walk-in sale, handed over at once (never in the pickup queue).
func (s *comboService) CounterSell(ctx context.Context, req dto.CounterComboOrderRequest) (*dto.ComboOrderResponse, error) {
	items, total, err := s.buildOrderItems(ctx, req.Items)
	if err != nil {
		return nil, err
	}

	payMethod := strings.TrimSpace(req.PayMethod)
	name := strings.TrimSpace(req.CustomerName)
	order := &models.ComboOrder{
		Status:      models.ComboOrderCollected,
		Total:       total,
		SoldChannel: models.SoldChannelCounter,
		PayMethod:   &payMethod,
	}
	if name != "" {
		order.CustomerName = &name
	}
	if err := s.orders.CreateCounterOrder(ctx, order, items); err != nil {
		return nil, err
	}
	// Best-effort audit: the order row is the source of truth.
	if rec, ok := audit.FromContext(ctx); ok {
		rec.ResourceID = order.ID
		rec.After = map[string]any{
			"status": models.ComboOrderCollected, "total": total,
			"sold_channel": models.SoldChannelCounter, "pay_method": payMethod,
		}
		if err := audit.In(ctx, s.db, rec); err != nil {
			logger.Warn("counter combo sale not audited", logger.String("order_id", order.ID), logger.Err(err))
		}
	}

	result := dto.NewComboOrderResponse(order, items)
	return &result, nil
}

var ictZone = time.FixedZone("ICT", 7*3600)

// PendingPickups is the counter handover board for one show date.
func (s *comboService) PendingPickups(ctx context.Context, date, search string) ([]dto.ComboPickupResponse, error) {
	day := time.Now().In(ictZone)
	if strings.TrimSpace(date) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(date), ictZone)
		if err != nil {
			return nil, apperrors.Validation("date must follow format YYYY-MM-DD")
		}
		day = parsed
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, ictZone)
	to := from.AddDate(0, 0, 1)

	rows, err := s.orders.PendingPickups(ctx, from, to, search, 100)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OrderID)
	}
	itemsByOrder, err := s.orders.ItemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ComboPickupResponse, 0, len(rows))
	for _, r := range rows {
		items := make([]dto.ComboOrderItemResponse, 0)
		for _, it := range itemsByOrder[r.OrderID] {
			items = append(items, dto.ComboOrderItemResponse{
				ComboID: it.ComboID, ComboName: it.ComboName,
				Quantity: it.Quantity, UnitPrice: it.UnitPrice,
				Subtotal: it.UnitPrice * int64(it.Quantity),
			})
		}
		row := dto.ComboPickupResponse{
			OrderID: r.OrderID, CustomerName: r.CustomerName, CustomerMail: r.CustomerMail,
			Total: r.Total, Items: items,
		}
		if r.BookingID != nil {
			row.BookingID = *r.BookingID
		}
		if r.MovieTitle != nil {
			row.MovieTitle = *r.MovieTitle
		}
		row.ShowtimeAt = r.ShowtimeAt
		out = append(out, row)
	}
	return out, nil
}

// CollectOrder hands one pre-order over; repeats read as already handed over.
func (s *comboService) CollectOrder(ctx context.Context, id string) (*dto.ComboOrderResponse, error) {
	won, err := s.orders.CollectCAS(ctx, id)
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, apperrors.NotFound("combo order already collected or missing")
	}
	order, err := s.orders.FindOrderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, apperrors.NotFound("combo order not found")
	}
	itemsByOrder, err := s.orders.ItemsByOrderIDs(ctx, []string{id})
	if err != nil {
		return nil, err
	}
	if rec, ok := audit.FromContext(ctx); ok {
		rec.ResourceID = id
		rec.After = map[string]any{"status": models.ComboOrderCollected}
		if err := audit.In(ctx, s.db, rec); err != nil {
			logger.Warn("combo collect not audited", logger.String("order_id", id), logger.Err(err))
		}
	}
	result := dto.NewComboOrderResponse(order, itemsByOrder[id])
	return &result, nil
}

// buildOrderItems prices lines from the live catalogue (all-or-nothing).
func (s *comboService) buildOrderItems(ctx context.Context, reqItems []dto.ComboOrderItemRequest) ([]models.ComboOrderItem, int64, error) {
	ids := make([]string, 0, len(reqItems))
	qtyByCombo := make(map[string]int, len(reqItems))
	for _, item := range reqItems {
		if item.Quantity <= 0 {
			return nil, 0, apperrors.ErrComboOrderEmpty
		}
		if _, ok := qtyByCombo[item.ComboID]; !ok {
			ids = append(ids, item.ComboID)
		}
		qtyByCombo[item.ComboID] += item.Quantity
	}

	found, err := s.combos.FindActiveByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	if len(found) != len(ids) {
		return nil, 0, apperrors.ErrComboInactive
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
		return nil, 0, apperrors.ErrComboOrderEmpty
	}
	return items, total, nil
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

// AdminList includes INACTIVE products (public ListActive never does).
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
		// Default on sale unless told otherwise.
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

	// Map write so price 0 / active false are stored, not skipped as zero values.
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

// AdminDelete soft-deletes so past receipts stay readable.
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

// comboAuditFields: plain map so audit never leaks future model fields.
func comboAuditFields(c *models.Combo) map[string]any {
	return map[string]any{
		"name":   c.Name,
		"price":  c.Price,
		"active": c.Active,
	}
}

// auditCombo is no-op without middleware audit in context.
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
