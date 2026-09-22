package repository

import (
	"context"
	"errors"
	"fmt"

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

// UpdateSeatSpan: the only write path for col_span (UpdateSeat can't touch it, so BulkUpdateSeats
// never changes a span by accident). MergeSeats/SplitSeat use it; must run in a transaction.
func (r *HallRepository) UpdateSeatSpan(tx *gorm.DB, seat *models.Seat) error {
	return tx.Model(seat).Select("seat_type", "col_span", "updated_at").Updates(seat).Error
}

// LockSchedule: hall scheduling lock (same key as ShowtimeRepository.LockHall), then showtime rows in id order.
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

// HallEverHadBooking: any booking of any status, incl. deleted showtimes (their seats/tickets
// still reference showtime_seats, so regenerating would violate the FK).
func (r *HallRepository) HallEverHadBooking(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM bookings b
		JOIN showtimes s ON s.id = b.showtime_id
		WHERE s.hall_id = ? LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// SeatEverHadBooking: per-seat version of HallEverHadBooking, gating DeleteRow/MergeSeats/SplitSeat
// (the whole-hall gate would refuse any hall that ever sold a ticket).
func (r *HallRepository) SeatEverHadBooking(tx *gorm.DB, seatID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM booking_seats bks
		JOIN showtime_seats ss ON ss.id = bks.showtime_seat_id
		WHERE ss.seat_id = ?
		UNION ALL
		SELECT 1 FROM tickets t
		JOIN showtime_seats ss ON ss.id = t.showtime_seat_id
		WHERE ss.seat_id = ? LIMIT 1`, seatID, seatID).
		Scan(&count).Error
	return count > 0, err
}

// HasOpenUpcomingShowtimes: open showtime still to come; checked before deactivating a hall.
func (r *HallRepository) HasOpenUpcomingShowtimes(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM showtimes
		WHERE hall_id = ? AND deleted_at IS NULL AND status = 'open' AND start_at > NOW() LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// HasUnfinishedShowtimes: showtime not yet ended (open or closed); checked before deleting a hall.
func (r *HallRepository) HasUnfinishedShowtimes(tx *gorm.DB, hallID string) (bool, error) {
	var count int64
	err := tx.Raw(`SELECT 1 FROM showtimes
		WHERE hall_id = ? AND deleted_at IS NULL AND end_at > NOW() LIMIT 1`, hallID).
		Scan(&count).Error
	return count > 0, err
}

// UpdateHall persists name/screen/aisle/active in a transaction. Whitelist excludes rows/
// seats_per_row so PUT /admin/halls/:id can never resize a hall; layout writes use UpdateHallLayout.
func (r *HallRepository) UpdateHall(tx *gorm.DB, hall *models.Hall) error {
	return tx.Model(hall).
		Select("name", "screen_position", "aisle_after_cols", "active", "updated_at").
		Updates(hall).Error
}

// UpdateHallLayout: the ONLY write path for rows/seats_per_row (UpdateHall's whitelist drops
// them — regenerating via UpdateHall once left `halls` describing a grid that no longer exists).
// Runs in a transaction after seats are written; name/active stay out (not a rename/reactivation).
func (r *HallRepository) UpdateHallLayout(tx *gorm.DB, hall *models.Hall) error {
	return tx.Model(hall).
		Select("rows", "seats_per_row", "screen_position", "aisle_after_cols", "updated_at").
		Updates(hall).Error
}

// DeleteHall soft-deletes and deactivates together; must run in a transaction.
func (r *HallRepository) DeleteHall(tx *gorm.DB, hallID string) error {
	if err := tx.Model(&models.Hall{}).Where("id = ?", hallID).Update("active", false).Error; err != nil {
		return err
	}
	return tx.Delete(&models.Hall{}, "id = ?", hallID).Error
}

// DeleteSeats drops a hall's seats before regenerate; only if HallEverHadBooking is false and
// after DeleteShowtimeSeatsByHall (FK). Must run in a transaction.
func (r *HallRepository) DeleteSeats(tx *gorm.DB, hallID string) error {
	return tx.Exec(`DELETE FROM seats WHERE hall_id = ?`, hallID).Error
}

// DeleteShowtimeSeatsByHall drops a hall's per-showtime seat state (deleted showtimes incl.),
// clearing the FK into seats. Only if HallEverHadBooking is false; must run in a transaction.
func (r *HallRepository) DeleteShowtimeSeatsByHall(tx *gorm.DB, hallID string) error {
	return tx.Exec(`DELETE FROM showtime_seats WHERE showtime_id IN
		(SELECT id FROM showtimes WHERE hall_id = ?)`, hallID).Error
}

// DeleteSeat drops one seat (DeleteRow, MergeSeats); only if SeatEverHadBooking is false and
// after DeleteShowtimeSeatsBySeat (FK). Must run in a transaction.
func (r *HallRepository) DeleteSeat(tx *gorm.DB, seatID string) error {
	return tx.Exec(`DELETE FROM seats WHERE id = ?`, seatID).Error
}

// DeleteShowtimeSeatsBySeat drops one seat's per-showtime state, clearing the FK before DeleteSeat.
// Only if SeatEverHadBooking is false; must run in a transaction.
func (r *HallRepository) DeleteShowtimeSeatsBySeat(tx *gorm.DB, seatID string) error {
	return tx.Exec(`DELETE FROM showtime_seats WHERE seat_id = ?`, seatID).Error
}

// RenumberRow shifts one row onto a new index/label for DeleteRow (rows stay 1..N).
// Never touches seat ids (FK-safe); runs in a transaction after the target row is gone.
func (r *HallRepository) RenumberRow(tx *gorm.DB, hallID string, fromIndex, toIndex int, toLabel string) error {
	return tx.Exec(`UPDATE seats SET row_index = ?, row_label = ?, updated_at = NOW()
		WHERE hall_id = ? AND row_index = ?`, toIndex, toLabel, hallID, fromIndex).Error
}

// CreateShowtimeSeatsForHall re-creates showtime_seats for the hall's open-to-come showtimes.
// In a transaction, after DeleteShowtimeSeatsByHall.
func (r *HallRepository) CreateShowtimeSeatsForHall(tx *gorm.DB, hallID string) error {
	return tx.Exec(`INSERT INTO showtime_seats (id, showtime_id, seat_id, status)
		SELECT gen_random_uuid(), st.id, se.id, 'available'
		FROM showtimes st, seats se
		WHERE st.hall_id = ? AND st.deleted_at IS NULL AND se.hall_id = ?`, hallID, hallID).Error
}

// CreateShowtimeSeatsForSeats: CreateShowtimeSeatsForHall scoped by seat id (incremental adds
// don't retouch other rows). In a transaction, after CreateSeats.
func (r *HallRepository) CreateShowtimeSeatsForSeats(tx *gorm.DB, hallID string, seatIDs []string) error {
	if len(seatIDs) == 0 {
		return nil
	}
	return tx.Exec(`INSERT INTO showtime_seats (id, showtime_id, seat_id, status)
		SELECT gen_random_uuid(), st.id, se.id, 'available'
		FROM showtimes st, seats se
		WHERE st.hall_id = ? AND st.deleted_at IS NULL AND se.id IN ?`, hallID, seatIDs).Error
}

func (r *HallRepository) PricesByHall(ctx context.Context, hallID string) ([]models.HallPrice, error) {
	var prices []models.HallPrice
	err := r.db.WithContext(ctx).Where("hall_id = ?", hallID).Order("seat_type").Find(&prices).Error
	return prices, err
}

// PublicPriceRow is one hall/seat-type price for the public price list.
type PublicPriceRow struct {
	HallID   string `gorm:"column:hall_id"`
	HallName string `gorm:"column:hall_name"`
	SeatType string `gorm:"column:seat_type"`
	Price    int64  `gorm:"column:price"`
}

// PublicPriceList returns every ACTIVE hall's seat-type prices in one query.
//
// It mirrors the gate the customer showtime query uses: a hall missing one of the
// four seat-type prices never reaches a customer, so listing it on a price page
// would advertise a hall nobody can book. `price > 0` for the same reason — 0
// means NOT CONFIGURED in this schema, not free.
func (r *HallRepository) PublicPriceList(ctx context.Context) ([]PublicPriceRow, error) {
	var rows []PublicPriceRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT h.id AS hall_id, h.name AS hall_name, hp.seat_type, hp.price
		FROM halls h
		JOIN hall_prices hp ON hp.hall_id = h.id
		WHERE h.deleted_at IS NULL AND h.active = TRUE AND hp.price > 0
		  AND h.id IN (
		      SELECT hall_id FROM hall_prices WHERE price > 0
		      GROUP BY hall_id HAVING count(DISTINCT seat_type) = 4
		  )
		ORDER BY h.name, hp.seat_type`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list public prices: %w", err)
	}
	return rows, nil
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
