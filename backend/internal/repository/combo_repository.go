package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// ComboRepository serves the combo/concession catalog (products), separate
// from ComboOrderRepository which stores what customers bought.
type ComboRepository interface {
	// ListActive returns every combo currently on sale, name-ordered.
	ListActive(ctx context.Context) ([]models.Combo, error)
	FindByID(ctx context.Context, id string) (*models.Combo, error)
	// FindActiveByIDs is used to price and validate an order's items in one
	// round trip; combos that are missing, soft-deleted or inactive are simply
	// absent from the returned map.
	FindActiveByIDs(ctx context.Context, ids []string) (map[string]models.Combo, error)
}

type comboRepository struct {
	db *gorm.DB
}

func NewComboRepository(db *gorm.DB) ComboRepository {
	return &comboRepository{db: db}
}

func (r *comboRepository) ListActive(ctx context.Context) ([]models.Combo, error) {
	var combos []models.Combo
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("name").Find(&combos).Error; err != nil {
		return nil, fmt.Errorf("list active combos: %w", err)
	}
	return combos, nil
}

func (r *comboRepository) FindByID(ctx context.Context, id string) (*models.Combo, error) {
	var combo models.Combo
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&combo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find combo: %w", err)
	}
	return &combo, nil
}

func (r *comboRepository) FindActiveByIDs(ctx context.Context, ids []string) (map[string]models.Combo, error) {
	byID := make(map[string]models.Combo, len(ids))
	if len(ids) == 0 {
		return byID, nil
	}
	var combos []models.Combo
	if err := r.db.WithContext(ctx).Where("id IN ? AND active = ?", ids, true).Find(&combos).Error; err != nil {
		return nil, fmt.Errorf("find active combos: %w", err)
	}
	for _, c := range combos {
		byID[c.ID] = c
	}
	return byID, nil
}

// ComboOrderRepository stores combo/concession purchases. Every write here is
// its own, independent transaction: a combo order failing must never roll back
// or block a ticket booking.
type ComboOrderRepository interface {
	// Create inserts the order and its items atomically (its own transaction,
	// never the caller's).
	Create(ctx context.Context, order *models.ComboOrder, items []models.ComboOrderItem) error
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.ComboOrder, int64, error)
	// ItemsByOrderIDs batches the item lookup for a page of orders.
	ItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]models.ComboOrderItem, error)
}

type comboOrderRepository struct {
	db *gorm.DB
}

func NewComboOrderRepository(db *gorm.DB) ComboOrderRepository {
	return &comboOrderRepository{db: db}
}

func (r *comboOrderRepository) Create(ctx context.Context, order *models.ComboOrder, items []models.ComboOrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("create combo order: %w", err)
		}
		for i := range items {
			items[i].ComboOrderID = order.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("create combo order items: %w", err)
		}
		return nil
	})
}

func (r *comboOrderRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.ComboOrder, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.ComboOrder{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count combo orders: %w", err)
	}
	var orders []models.ComboOrder
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("list combo orders: %w", err)
	}
	return orders, total, nil
}

func (r *comboOrderRepository) ItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]models.ComboOrderItem, error) {
	out := make(map[string][]models.ComboOrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	var items []models.ComboOrderItem
	if err := r.db.WithContext(ctx).Where("combo_order_id IN ?", orderIDs).Order("created_at").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("find combo order items: %w", err)
	}
	for _, item := range items {
		out[item.ComboOrderID] = append(out[item.ComboOrderID], item)
	}
	return out, nil
}
