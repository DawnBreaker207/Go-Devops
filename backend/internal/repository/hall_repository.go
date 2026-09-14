package repository

import (
	"context"
	"errors"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HallRepository struct {
	db *gorm.DB
}

func NewHallRepository(db *gorm.DB) *HallRepository {
	return &HallRepository{db: db}
}

func (r *HallRepository) CreateHall(tx *gorm.DB, hall *models.Hall) error {
	return tx.Create(hall).Error
}

func (r *HallRepository) CreateSeats(tx *gorm.DB, seats []models.Seat) error {
	if len(seats) == 0 {
		return nil
	}
	return tx.Create(&seats).Error
}

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

func (r *HallRepository) SeatsByHall(ctx context.Context, hallID string) ([]models.Seat, error) {
	var seats []models.Seat
	err := r.db.WithContext(ctx).
		Where("hall_id = ?", hallID).Order("row_label, col_number").Find(&seats).Error
	return seats, err
}

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

func (r *HallRepository) UpdateSeat(tx *gorm.DB, seat *models.Seat) error {
	return tx.Model(seat).Select("seat_type", "is_gap", "updated_at").Updates(seat).Error
}

// LockSchedule takes the hall's scheduling lock (same key as ShowtimeRepository.LockHall), then
// locks the hall's showtime rows in id order, which holds and confirms share-lock.
func (r *HallRepository) LockSchedule(tx *gorm.DB, hallID string) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", hallID).Error; err != nil {
		return err
	}
	var ids []string
	return tx.Raw(`SELECT id FROM showtimes WHERE hall_id = ? AND deleted_at IS NULL ORDER BY id FOR UPDATE`, hallID).
		Scan(&ids).Error
}

// Expired and refunded bookings sold no seat, so they do not lock the layout.
func (r *HallRepository) HallHasBookings(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings b
		JOIN showtimes s ON s.id = b.showtime_id AND s.deleted_at IS NULL
		WHERE s.hall_id = ? AND b.status IN ('pending', 'confirmed') LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

func (r *HallRepository) PricesByHall(ctx context.Context, hallID string) ([]models.HallPrice, error) {
	var prices []models.HallPrice
	err := r.db.WithContext(ctx).Where("hall_id = ?", hallID).Order("seat_type").Find(&prices).Error
	return prices, err
}

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
