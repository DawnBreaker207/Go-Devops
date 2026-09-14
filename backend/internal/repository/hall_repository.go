package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HallRepository persists halls, seats and their prices.
type HallRepository struct {
	db *gorm.DB
}

func NewHallRepository(db *gorm.DB) *HallRepository {
	return &HallRepository{db: db}
}

// CreateHall inserts a hall. It must run inside a transaction.
func (r *HallRepository) CreateHall(tx *gorm.DB, hall *models.Hall) error {
	return tx.Create(hall).Error
}

// CreateSeats inserts the generated seat grid. It must run inside a transaction.
func (r *HallRepository) CreateSeats(tx *gorm.DB, seats []models.Seat) error {
	if len(seats) == 0 {
		return nil
	}
	return tx.Create(&seats).Error
}

// List returns halls matching the query, paginated by name.
func (r *HallRepository) List(ctx context.Context, query string, limit, offset int) ([]models.Hall, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Hall{}).
		Where("name ILIKE ?", "%"+query+"%").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var halls []models.Hall
	err := r.db.WithContext(ctx).
		Where("name ILIKE ?", "%"+query+"%").Order("name").
		Limit(limit).Offset(offset).Find(&halls).Error
	return halls, total, err
}

// FindByID loads one hall.
func (r *HallRepository) FindByID(ctx context.Context, id string) (*models.Hall, error) {
	var hall models.Hall
	if err := r.db.WithContext(ctx).First(&hall, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &hall, nil
}

// SeatsByHall returns the seat grid of a hall, ordered by row and column.
func (r *HallRepository) SeatsByHall(ctx context.Context, hallID string) ([]models.Seat, error) {
	var seats []models.Seat
	err := r.db.WithContext(ctx).
		Where("hall_id = ?", hallID).Order("row_label, col_number").Find(&seats).Error
	return seats, err
}

// FindSeat loads one seat belonging to the hall.
func (r *HallRepository) FindSeat(ctx context.Context, hallID, seatID string) (*models.Seat, error) {
	var seat models.Seat
	if err := r.db.WithContext(ctx).
		First(&seat, "hall_id = ? AND id = ?", hallID, seatID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &seat, nil
}

// UpdateSeat persists a seat. It must run inside a transaction.
func (r *HallRepository) UpdateSeat(tx *gorm.DB, seat *models.Seat) error {
	return tx.Model(seat).Select("seat_type", "is_gap", "updated_at").Updates(seat).Error
}

// HallHasBookings reports whether any booking sits on a showtime in the hall.
// Uses a raw query because the bookings model is not kept in the codebase.
func (r *HallRepository) HallHasBookings(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings b
		JOIN showtimes s ON s.id = b.showtime_id AND s.deleted_at IS NULL
		WHERE s.hall_id = ? AND b.deleted_at IS NULL LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// PricesByHall returns all seat-type prices of a hall.
func (r *HallRepository) PricesByHall(ctx context.Context, hallID string) ([]models.HallPrice, error) {
	var prices []models.HallPrice
	err := r.db.WithContext(ctx).Where("hall_id = ?", hallID).Order("seat_type").Find(&prices).Error
	return prices, err
}

// UpsertPrices inserts or updates the four seat-type prices of a hall.
// It must run inside a transaction.
func (r *HallRepository) UpsertPrices(tx *gorm.DB, hallID string, prices map[string]int64) error {
	rows := make([]models.HallPrice, 0, len(prices))
	for seatType, price := range prices {
		rows = append(rows, models.HallPrice{
			HallID:   hallID,
			SeatType: seatType,
			Price:    price,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "hall_id"}, {Name: "seat_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"price", "updated_at"}),
	}).Create(&rows).Error
}