package service

import (
	"context"

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
}

type comboService struct {
	combos      repository.ComboRepository
	orders      repository.ComboOrderRepository
	bookingRepo repository.BookingRepository // optional ownership check only; may be nil
}

// NewComboService: bookingRepo may be nil, which simply skips the ownership
// check when an order names a booking_id.
func NewComboService(combos repository.ComboRepository, orders repository.ComboOrderRepository, bookingRepo repository.BookingRepository) ComboService {
	return &comboService{combos: combos, orders: orders, bookingRepo: bookingRepo}
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
