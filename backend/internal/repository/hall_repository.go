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
		Where("hall_id = ?", hallID).Order("row_index, col_number").Find(&seats).Error
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

// HallEverHadBooking reports any booking of any status, including a
// showtime later deleted: its booking_seats/tickets still reference the
// hall's showtime_seats rows by foreign key, so regenerating the layout
// (which deletes and recreates them) would fail, not just race.
func (r *HallRepository) HallEverHadBooking(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings b
		JOIN showtimes s ON s.id = b.showtime_id
		WHERE s.hall_id = ? LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// HasOpenUpcomingShowtimes reports an open showtime still to come, checked
// before deactivating a hall.
func (r *HallRepository) HasOpenUpcomingShowtimes(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM showtimes
		WHERE hall_id = ? AND deleted_at IS NULL AND status = 'open' AND start_at > NOW() LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// HasUnfinishedShowtimes reports any showtime that has not ended yet
// (open or closed), checked before deleting a hall.
func (r *HallRepository) HasUnfinishedShowtimes(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM showtimes
		WHERE hall_id = ? AND deleted_at IS NULL AND end_at > NOW() LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// UpdateHall persists name/screen/aisle/active. It must run inside a transaction.
func (r *HallRepository) UpdateHall(tx *gorm.DB, hall *models.Hall) error {
	return tx.Model(hall).
		Select("name", "screen_position", "aisle_after_cols", "active", "updated_at").
		Updates(hall).Error
}

// DeleteHall soft-deletes a hall, also turning off active so the two flags
// never disagree forever (a deleted hall is never "still active"). Must run
// inside a transaction.
func (r *HallRepository) DeleteHall(tx *gorm.DB, hallID string) error {
	if err := tx.Model(&models.Hall{}).Where("id = ?", hallID).Update("active", false).Error; err != nil {
		return err
	}
	return tx.Delete(&models.Hall{}, "id = ?", hallID).Error
}

// DeleteSeats drops every seat of a hall before its layout is regenerated;
// only legal when HallEverHadBooking is false, and only after
// DeleteShowtimeSeatsByHall (showtime_seats.seat_id would otherwise block
// the delete). It must run inside a transaction.
func (r *HallRepository) DeleteSeats(tx *gorm.DB, hallID string) error {
	return tx.Exec(`DELETE FROM seats WHERE hall_id = ?`, hallID).Error
}

// DeleteShowtimeSeatsByHall drops the per-seat state of every showtime of the
// hall (deleted or not), clearing the foreign key into seats before the grid
// is rebuilt. Only legal when HallEverHadBooking is false: a booking_seats or
// tickets row would otherwise still reference these rows and block this
// delete. It must run inside a transaction.
func (r *HallRepository) DeleteShowtimeSeatsByHall(tx *gorm.DB, hallID string) error {
	return tx.Exec(`DELETE FROM showtime_seats WHERE showtime_id IN
		(SELECT id FROM showtimes WHERE hall_id = ?)`, hallID).Error
}

// CreateShowtimeSeatsForHall re-creates showtime_seats for every seat of the
// hall's own showtimes still open to come, from the grid just written by
// CreateSeats. It must run inside a transaction, after DeleteShowtimeSeatsByHall.
func (r *HallRepository) CreateShowtimeSeatsForHall(tx *gorm.DB, hallID string) error {
	return tx.Exec(`INSERT INTO showtime_seats (id, showtime_id, seat_id, status)
		SELECT gen_random_uuid(), st.id, se.id, 'available'
		FROM showtimes st, seats se
		WHERE st.hall_id = ? AND st.deleted_at IS NULL AND se.hall_id = ?`, hallID, hallID).Error
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
