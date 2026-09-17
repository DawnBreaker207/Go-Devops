package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type ComboRepository struct {
	db *gorm.DB
}

func NewComboRepository(db *gorm.DB) *ComboRepository {
	return &ComboRepository{db: db}
}

func (r *ComboRepository) Create(tx *gorm.DB, c *models.Combo) error {
	return tx.Create(c).Error
}

func (r *ComboRepository) List(ctx context.Context, activeOnly bool, limit, offset int) ([]models.Combo, int64, error) {
	q := r.db.WithContext(ctx).Model(&models.Combo{})
	if activeOnly {
		q = q.Where("active = true")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var combos []models.Combo
	err := q.Order("name").Limit(limit).Offset(offset).Find(&combos).Error
	return combos, total, err
}

func (r *ComboRepository) FindByID(ctx context.Context, id string) (*models.Combo, error) {
	var c models.Combo
	if err := r.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// FindByIDs is used both at hold time inside the booking transaction (tx set)
// and for display outside one (tx nil).
func (r *ComboRepository) FindByIDs(ctx context.Context, tx *gorm.DB, ids []string) (map[string]models.Combo, error) {
	if len(ids) == 0 {
		return map[string]models.Combo{}, nil
	}
	conn := r.db.WithContext(ctx)
	if tx != nil {
		conn = tx
	}
	var combos []models.Combo
	if err := conn.Where("id IN ?", ids).Find(&combos).Error; err != nil {
		return nil, err
	}
	byID := make(map[string]models.Combo, len(combos))
	for _, c := range combos {
		byID[c.ID] = c
	}
	return byID, nil
}

func (r *ComboRepository) Update(tx *gorm.DB, c *models.Combo) error {
	return tx.Model(c).Select("name", "description", "price", "member_price", "active", "updated_at").Updates(c).Error
}

// SetStock upserts a (branch, combo) cap; admin-only, manual (Phần 2.2 "Nhập/bổ sung kho: admin cập nhật tay").
func (r *ComboRepository) SetStock(tx *gorm.DB, branchID, comboID string, quantity int) error {
	return tx.Exec(`INSERT INTO combo_branch_stock (branch_id, combo_id, stock_quantity, updated_at)
		VALUES (?, ?, ?, NOW())
		ON CONFLICT (branch_id, combo_id) DO UPDATE SET stock_quantity = EXCLUDED.stock_quantity, updated_at = NOW()`,
		branchID, comboID, quantity).Error
}

// ReserveStock is a no-op (unlimited) when no combo_branch_stock row exists
// for this (branch, combo); otherwise it is the CAS that reserves qty units,
// ok=false meaning genuinely out of stock (Phần 2.2).
func (r *ComboRepository) ReserveStock(tx *gorm.DB, branchID, comboID string, qty int) (ok bool, err error) {
	var exists bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM combo_branch_stock WHERE branch_id = ? AND combo_id = ?)`,
		branchID, comboID).Scan(&exists).Error; err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}
	res := tx.Exec(`UPDATE combo_branch_stock SET stock_quantity = stock_quantity - ?, updated_at = NOW()
		WHERE branch_id = ? AND combo_id = ? AND stock_quantity >= ?`, qty, branchID, comboID, qty)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ReleaseStock adds qty back; a no-op if no stock row exists for this
// (branch, combo) — matches ReserveStock's unlimited case.
func (r *ComboRepository) ReleaseStock(tx *gorm.DB, branchID, comboID string, qty int) error {
	return tx.Exec(`UPDATE combo_branch_stock SET stock_quantity = stock_quantity + ?, updated_at = NOW()
		WHERE branch_id = ? AND combo_id = ?`, qty, branchID, comboID).Error
}

// ReleaseStockForBooking releases every combo line of a booking back to
// stock — called when a booking that reserved combo stock at hold does not
// end up CONFIRMED (Phần 2.2, same rollback pattern as vouchers).
func (r *ComboRepository) ReleaseStockForBooking(ctx context.Context, tx *gorm.DB, branchID, bookingID string) error {
	lines, err := r.BookingCombos(ctx, tx, bookingID)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if err := r.ReleaseStock(tx, branchID, l.ComboID, l.Quantity); err != nil {
			return err
		}
	}
	return nil
}

func (r *ComboRepository) CreateBookingCombos(tx *gorm.DB, rows []models.BookingCombo) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ComboRepository) BookingCombos(ctx context.Context, tx *gorm.DB, bookingID string) ([]models.BookingCombo, error) {
	conn := r.db.WithContext(ctx)
	if tx != nil {
		conn = tx
	}
	var rows []models.BookingCombo
	err := conn.Where("booking_id = ?", bookingID).Find(&rows).Error
	return rows, err
}

// MarkDelivered is the counter pickup flow: staff looks a booking up by its
// code, sees its combo lines, and marks each one delivered once, not twice.
func (r *ComboRepository) MarkDelivered(tx *gorm.DB, bookingID, comboID string) (int64, error) {
	res := tx.Model(&models.BookingCombo{}).
		Where("booking_id = ? AND combo_id = ? AND delivered = false", bookingID, comboID).
		Updates(map[string]any{"delivered": true, "delivered_at": gorm.Expr("NOW()")})
	return res.RowsAffected, res.Error
}
