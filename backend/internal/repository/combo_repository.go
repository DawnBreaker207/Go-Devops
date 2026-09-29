package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// ComboRepository serves the combo/concession catalog (products), separate
// from ComboOrderRepository which stores what customers bought.
type ComboRepository interface {
	// ListActive returns every combo currently on sale, name-ordered.
	ListActive(ctx context.Context) ([]models.Combo, error)
	FindByID(ctx context.Context, id string) (*models.Combo, error)
// FindActiveByIDs prices/validates order items in one trip; missing/inactive
// combos are simply absent from map.
	FindActiveByIDs(ctx context.Context, ids []string) (map[string]models.Combo, error)

// List is operator view: paged, search, shows INACTIVE too. nil active = no filter.
	List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.Combo, int64, error)
// The three writes take caller tx so service puts audit row in same tx.
	Create(ctx context.Context, tx *gorm.DB, combo *models.Combo) error
	// Update writes only the columns named in `fields`, so a partial request can
	// never blank the columns it did not mention.
	Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error
	// SoftDelete keeps past combo_order_items readable (they snapshot name and
	// price, but the FK still points here).
	SoftDelete(ctx context.Context, tx *gorm.DB, id string) error
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

func (r *comboRepository) List(ctx context.Context, page, pageSize int, search string, active *bool) ([]models.Combo, int64, error) {
	// Built twice (count + page) from one filter helper so the two can never drift.
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
	if err := filter(r.db.WithContext(ctx).Model(&models.Combo{})).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count combos: %w", err)
	}

	var combos []models.Combo
	if err := filter(r.db.WithContext(ctx)).
		Order("name").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&combos).Error; err != nil {
		return nil, 0, fmt.Errorf("list combos: %w", err)
	}
	return combos, total, nil
}

func (r *comboRepository) Create(ctx context.Context, tx *gorm.DB, combo *models.Combo) error {
	// Writes Active exactly as given, including false — see the comment on
	// models.Combo.Active for why that field carries no `default:` tag.
	if err := tx.WithContext(ctx).Create(combo).Error; err != nil {
		return fmt.Errorf("create combo: %w", err)
	}
	return nil
}

func (r *comboRepository) Update(ctx context.Context, tx *gorm.DB, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	// Model(&Combo{}) + Where, not Updates(struct): a map write is the only way
	// to set a column to its zero value (price 0, active false) on purpose.
	if err := tx.WithContext(ctx).Model(&models.Combo{}).
		Where("id = ?", id).Updates(fields).Error; err != nil {
		return fmt.Errorf("update combo: %w", err)
	}
	return nil
}

func (r *comboRepository) SoftDelete(ctx context.Context, tx *gorm.DB, id string) error {
	if err := tx.WithContext(ctx).Where("id = ?", id).Delete(&models.Combo{}).Error; err != nil {
		return fmt.Errorf("delete combo: %w", err)
	}
	return nil
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

// ComboOrderRepository stores purchases; each write is own tx so combo failure
// never rolls back or blocks ticket booking.
type ComboOrderRepository interface {
	// Create inserts the order and its items atomically (its own transaction,
	// never the caller's).
	Create(ctx context.Context, order *models.ComboOrder, items []models.ComboOrderItem) error
	// CreateCounterOrder: walk-in sale with explicit columns (user_id NULL).
	CreateCounterOrder(ctx context.Context, order *models.ComboOrder, items []models.ComboOrderItem) error
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.ComboOrder, int64, error)
	FindOrderByID(ctx context.Context, id string) (*models.ComboOrder, error)
	// ItemsByOrderIDs batches the item lookup for a page of orders.
	ItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]models.ComboOrderItem, error)
	// PendingPickups lists online confirmed orders awaiting handover.
	PendingPickups(ctx context.Context, from, to time.Time, search string, limit int) ([]PendingPickupRow, error)
	// CollectCAS flips confirmed -> collected exactly once.
	CollectCAS(ctx context.Context, id string) (bool, error)
}

// PendingPickupRow is one line on the counter's handover board.
type PendingPickupRow struct {
	OrderID      string
	CustomerName string
	CustomerMail string
	BookingID    *string
	MovieTitle   *string
	ShowtimeAt   *time.Time
	Total        int64
	CreatedAt    time.Time
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

func (r *comboOrderRepository) CreateCounterOrder(ctx context.Context, order *models.ComboOrder, items []models.ComboOrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if order.ID == "" {
			order.ID = uuid.NewString()
		}
		if err := tx.Exec(`INSERT INTO combo_orders
			(id, user_id, booking_id, status, total, sold_channel, pay_method, customer_name, created_at, updated_at)
			VALUES (?, NULL, NULL, ?, ?, ?, ?, NULLIF(?, ''), NOW(), NOW())`,
			order.ID, order.Status, order.Total, order.SoldChannel, order.PayMethod, order.CustomerName).Error; err != nil {
			return fmt.Errorf("create counter combo order: %w", err)
		}
		for i := range items {
			items[i].ComboOrderID = order.ID
		}
		if err := tx.Create(&items).Error; err != nil {
			return fmt.Errorf("create counter combo order items: %w", err)
		}
		return nil
	})
}

func (r *comboOrderRepository) PendingPickups(ctx context.Context, from, to time.Time, search string, limit int) ([]PendingPickupRow, error) {
	var rows []PendingPickupRow
	q := r.db.WithContext(ctx).Table("combo_orders o").
		Select(`o.id AS order_id, COALESCE(u.full_name, '') AS customer_name, COALESCE(u.email, '') AS customer_mail,
			o.booking_id, m.title AS movie_title, st.start_at AS showtime_at, o.total, o.created_at`).
		Joins("JOIN users u ON u.id = o.user_id").
		Joins("LEFT JOIN bookings b ON b.id = o.booking_id").
		Joins("LEFT JOIN showtimes st ON st.id = b.showtime_id").
		Joins("LEFT JOIN movies m ON m.id = st.movie_id").
		Where("o.status = ? AND o.sold_channel = ?", models.ComboOrderConfirmed, models.SoldChannelOnline).
		Where("(o.booking_id IS NULL OR (st.start_at >= ? AND st.start_at < ?))", from, to).
		Order("o.created_at")
	if s := strings.TrimSpace(search); s != "" {
		like := "%" + s + "%"
		q = q.Where(`(o.id::text ILIKE ? OR u.full_name ILIKE ? OR u.email ILIKE ? OR EXISTS
			(SELECT 1 FROM tickets t WHERE t.booking_id = o.booking_id AND t.code ILIKE ?))`,
			like, like, like, like)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("pending pickups: %w", err)
	}
	return rows, nil
}

func (r *comboOrderRepository) CollectCAS(ctx context.Context, id string) (bool, error) {
	res := r.db.WithContext(ctx).Table("combo_orders").
		Where("id = ? AND status = ?", id, models.ComboOrderConfirmed).
		Update("status", models.ComboOrderCollected)
	if res.Error != nil {
		return false, fmt.Errorf("collect combo order: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r *comboOrderRepository) FindOrderByID(ctx context.Context, id string) (*models.ComboOrder, error) {
	var order models.ComboOrder
	if err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find combo order: %w", err)
	}
	return &order, nil
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
