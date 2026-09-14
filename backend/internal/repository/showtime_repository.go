package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ShowtimeRow is a showtime joined with its hall name and movie title.
type ShowtimeRow struct {
	models.Showtime
	HallName    string `gorm:"column:hall_name"`
	MovieTitle  string `gorm:"column:movie_title"`
	MovieStatus string `gorm:"column:movie_status"`
}

// ShowtimePickRow is a showtime in the customer picker, with price.
type ShowtimePickRow struct {
	ID         string    `gorm:"column:id"`
	MovieID    string    `gorm:"column:movie_id"`
	MovieTitle string    `gorm:"column:movie_title"`
	HallID     string    `gorm:"column:hall_id"`
	HallName   string    `gorm:"column:hall_name"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
	Status     string    `gorm:"column:status"`
	FromPrice  int64     `gorm:"column:from_price"`
}

// SeatMapRow is a seat of a showtime with its state and price.
type SeatMapRow struct {
	ShowtimeID     string    `gorm:"column:showtime_id"`
	SeatShowtimeID string    `gorm:"column:showtime_seat_id"`
	MovieID        string    `gorm:"column:movie_id"`
	MovieTitle     string    `gorm:"column:movie_title"`
	HallID         string    `gorm:"column:hall_id"`
	HallName       string    `gorm:"column:hall_name"`
	StartAt        time.Time `gorm:"column:start_at"`
	EndAt          time.Time `gorm:"column:end_at"`
	Status         string    `gorm:"column:status"`
	SeatID         string    `gorm:"column:seat_id"`
	RowLabel       string    `gorm:"column:row_label"`
	ColNumber      int       `gorm:"column:col_number"`
	SeatType       string    `gorm:"column:seat_type"`
	IsGap          bool      `gorm:"column:is_gap"`
	SeatStatus     string    `gorm:"column:seat_status"`
	Price          int64     `gorm:"column:price"`
}

// ShowtimeRepository persists showtimes and showtime seats.
type ShowtimeRepository struct {
	db *gorm.DB
}

func NewShowtimeRepository(db *gorm.DB) *ShowtimeRepository {
	return &ShowtimeRepository{db: db}
}

// LockHall serialises showtime scheduling for a hall.
func (r *ShowtimeRepository) LockHall(tx *gorm.DB, hallID string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", hallID).Error
}

// Create inserts a showtime. It must run inside a transaction.
func (r *ShowtimeRepository) Create(tx *gorm.DB, showtime *models.Showtime) error {
	return tx.Create(showtime).Error
}

// CreateSeatStates initialises one per-seat state row per hall seat so the
// grid is lockable from the first second. Idempotent per seat.
func (r *ShowtimeRepository) CreateSeatStates(tx *gorm.DB, showtimeID string, seats []models.Seat) error {
	if len(seats) == 0 {
		return nil
	}
	rows := make([]models.ShowtimeSeat, 0, len(seats))
	for _, seat := range seats {
		rows = append(rows, models.ShowtimeSeat{
			ShowtimeID: showtimeID,
			SeatID:     seat.ID,
			Status:     models.SeatStatusAvailable,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "showtime_id"}, {Name: "seat_id"}},
		DoNothing: true,
	}).Create(&rows).Error
}

// Update persists a showtime. It must run inside a transaction.
func (r *ShowtimeRepository) Update(tx *gorm.DB, showtime *models.Showtime) error {
	return tx.Model(showtime).
		Select("movie_id", "hall_id", "start_at", "end_at", "status", "updated_at").
		Updates(showtime).Error
}

// Delete soft-deletes a showtime. It must run inside a transaction.
func (r *ShowtimeRepository) Delete(tx *gorm.DB, showtimeID string) error {
	return tx.Delete(&models.Showtime{}, "id = ?", showtimeID).Error
}

// LockForUpdate locks a (not deleted) showtime row for an admin change. Holds
// and confirms share-lock the same row, so each side waits for the other and
// sees what the other committed.
func (r *ShowtimeRepository) LockForUpdate(tx *gorm.DB, id string) (*models.Showtime, error) {
	var showtime models.Showtime
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&showtime).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &showtime, nil
}

// DeleteSeatStates drops the seat grid of a showtime that never had a booking,
// before it is rebuilt for another hall.
func (r *ShowtimeRepository) DeleteSeatStates(tx *gorm.DB, showtimeID string) error {
	return tx.Exec(`DELETE FROM showtime_seats WHERE showtime_id = ?`, showtimeID).Error
}

// FindByID returns a showtime joined with hall name and movie title.
func (r *ShowtimeRepository) FindByID(ctx context.Context, id string) (*ShowtimeRow, error) {
	var row ShowtimeRow
	err := r.db.WithContext(ctx).
		Model(&models.Showtime{}).
		Select(`showtimes.id, showtimes.movie_id, showtimes.hall_id, showtimes.start_at,
			showtimes.end_at, showtimes.status, showtimes.created_at, showtimes.updated_at,
			halls.name AS hall_name, movies.title AS movie_title, movies.status AS movie_status`).
		Joins("JOIN halls ON halls.id = showtimes.hall_id").
		Joins("JOIN movies ON movies.id = showtimes.movie_id").
		Where("showtimes.id = ?", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// OverlapCount returns how many showtimes in the hall collide with one running
// [start, end]. The cleaning buffer applies on both sides: after the earlier
// showtime and before the later one (H5). excludeID is the showtime being
// changed (empty on create).
func (r *ShowtimeRepository) OverlapCount(tx *gorm.DB, hallID string, start, end time.Time, buffer time.Duration, excludeID string) (int64, error) {
	var count int64
	q := tx.Model(&models.Showtime{}).
		Where("hall_id = ? AND start_at < ? AND end_at > ?", hallID, end.Add(buffer), start.Add(-buffer))
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	err := q.Count(&count).Error
	return count, err
}

// ShowtimeHasBookings reports whether any booking references the showtime.
// Uses a raw query because the bookings model is not kept in the codebase.
func (r *ShowtimeRepository) ShowtimeHasBookings(tx *gorm.DB, showtimeID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings WHERE showtime_id = ? AND deleted_at IS NULL LIMIT 1`,
		showtimeID).Scan(&count).Error
	return count > 0, err
}

// ShowtimeHasLiveBookings reports whether a PENDING or CONFIRMED booking holds
// or bought seats of the showtime.
func (r *ShowtimeRepository) ShowtimeHasLiveBookings(tx *gorm.DB, showtimeID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings WHERE showtime_id = ? AND status IN ('pending', 'confirmed')
		AND deleted_at IS NULL LIMIT 1`, showtimeID).Scan(&count).Error
	return count > 0, err
}

// PickingList returns open, not yet started showtimes inside [start, end) of
// movies that are showing (one movie, or all when movieID is empty), only in
// halls that have a price for every seat type (FR-CAT-01..03).
func (r *ShowtimeRepository) PickingList(ctx context.Context, movieID string, start, end, now time.Time) ([]ShowtimePickRow, error) {
	var rows []ShowtimePickRow
	q := r.db.WithContext(ctx).
		Model(&models.Showtime{}).
		Select(`showtimes.id, showtimes.movie_id, movies.title AS movie_title, showtimes.hall_id,
			showtimes.start_at, showtimes.end_at, showtimes.status, halls.name AS hall_name,
			MIN(hall_prices.price) AS from_price`).
		Joins("JOIN movies ON movies.id = showtimes.movie_id AND movies.deleted_at IS NULL AND movies.status = ?", models.MovieStatusShowing).
		Joins("JOIN halls ON halls.id = showtimes.hall_id AND halls.deleted_at IS NULL").
		Joins("JOIN hall_prices ON hall_prices.hall_id = showtimes.hall_id")
	if movieID != "" {
		q = q.Where("showtimes.movie_id = ?", movieID)
	}
	err := q.
		Where("showtimes.deleted_at IS NULL").
		Where("showtimes.status = ?", models.ShowtimeOpen).
		Where("showtimes.start_at >= ? AND showtimes.start_at < ?", start, end).
		Where("showtimes.start_at >= ?", now).
		Where(`showtimes.hall_id IN (SELECT hall_id FROM hall_prices
			GROUP BY hall_id HAVING count(DISTINCT seat_type) = ?)`, len(models.AllSeatTypes)).
		Group("showtimes.id, movies.title, halls.name").
		Order("showtimes.start_at").
		Scan(&rows).Error
	return rows, err
}

// SeatMap returns every seat of a showtime with state and price.
func (r *ShowtimeRepository) SeatMap(ctx context.Context, showtimeID string) ([]SeatMapRow, error) {
	var rows []SeatMapRow
	err := r.db.WithContext(ctx).
		Model(&models.Showtime{}).
		Select(`showtimes.id AS showtime_id, showtimes.movie_id, showtimes.hall_id,
			showtimes.start_at, showtimes.end_at, showtimes.status,
			movies.title AS movie_title, halls.name AS hall_name,
			showtime_seats.id AS showtime_seat_id, seats.id AS seat_id,
			seats.row_label, seats.col_number,
			seats.seat_type, seats.is_gap,
			COALESCE(showtime_seats.status, '') AS seat_status,
			COALESCE(hall_prices.price, 0) AS price`).
		Joins("JOIN halls ON halls.id = showtimes.hall_id").
		Joins("JOIN movies ON movies.id = showtimes.movie_id").
		Joins("JOIN seats ON seats.hall_id = showtimes.hall_id").
		Joins("LEFT JOIN showtime_seats ON showtime_seats.showtime_id = showtimes.id AND showtime_seats.seat_id = seats.id").
		Joins("LEFT JOIN hall_prices ON hall_prices.hall_id = showtimes.hall_id AND hall_prices.seat_type = seats.seat_type").
		Where("showtimes.id = ?", showtimeID).
		Order("seats.row_label, seats.col_number").
		Scan(&rows).Error
	return rows, err
}
