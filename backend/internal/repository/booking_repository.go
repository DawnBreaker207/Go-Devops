package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type TicketRow struct {
	ID             string `gorm:"column:id"`
	BookingID      string `gorm:"column:booking_id"`
	ShowtimeSeatID string `gorm:"column:showtime_seat_id"`
	Price          int64  `gorm:"column:price"`
	Code           string `gorm:"column:code"`
	Status         string `gorm:"column:status"`
	RowLabel       string `gorm:"column:row_label"`
	ColNumber      int    `gorm:"column:col_number"`
	SeatType       string `gorm:"column:seat_type"`
}

type TicketGateRow struct {
	ID            string    `gorm:"column:id"`
	Code          string    `gorm:"column:code"`
	Status        string    `gorm:"column:status"`
	BookingStatus string    `gorm:"column:booking_status"`
	ShowtimeID    string    `gorm:"column:showtime_id"`
	StartAt       time.Time `gorm:"column:start_at"`
	HallName      string    `gorm:"column:hall_name"`
	MovieTitle    string    `gorm:"column:movie_title"`
	AgeRating     string    `gorm:"column:age_rating"`
	RowLabel      string    `gorm:"column:row_label"`
	ColNumber     int       `gorm:"column:col_number"`
}

type BookingHeader struct {
	ID          string    `gorm:"column:id"`
	Status      string    `gorm:"column:status"`
	TotalAmount int64     `gorm:"column:total_amount"`
	SoldVia     string    `gorm:"column:sold_via"`
	Email       string    `gorm:"column:email"`
	FullName    string    `gorm:"column:full_name"`
	MovieTitle  string    `gorm:"column:movie_title"`
	AgeRating   string    `gorm:"column:age_rating"`
	HallName    string    `gorm:"column:hall_name"`
	StartAt     time.Time `gorm:"column:start_at"`
}

type ShowtimeInfoRow struct {
	ID         string    `gorm:"column:id"`
	MovieID    string    `gorm:"column:movie_id"`
	MovieTitle string    `gorm:"column:movie_title"`
	AgeRating  string    `gorm:"column:age_rating"`
	HallID     string    `gorm:"column:hall_id"`
	HallName   string    `gorm:"column:hall_name"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
}

// Methods taking tx run in the caller's transaction; a nil tx uses the plain connection.
type BookingRepository interface {
	Now(ctx context.Context, tx *gorm.DB) (time.Time, error)

	FindByID(ctx context.Context, id string) (*models.Booking, error)
	FindByIDAndUser(ctx context.Context, id, userID string) (*models.Booking, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.Booking, int64, error)

	LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error
	UserActive(ctx context.Context, tx *gorm.DB, userID string) (bool, error)
	LockBooking(ctx context.Context, tx *gorm.DB, id string) (*models.Booking, error)
	LockLatestByKey(ctx context.Context, tx *gorm.DB, idempotencyKey string) (*models.Booking, error)
	LockPendingByUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) (*models.Booking, error)
	LockShowtime(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error)
	MovieShowing(ctx context.Context, tx *gorm.DB, movieID string) (bool, error)

	Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
	CreateCounterBooking(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
	CreateBookingSeats(ctx context.Context, tx *gorm.DB, seats []models.BookingSeat) error
	BookingSeats(ctx context.Context, tx *gorm.DB, bookingID string) ([]models.BookingSeat, error)

	ExpireBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	SetPaid(ctx context.Context, tx *gorm.DB, id, paymentID string) (int64, error)
	ConfirmBooking(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	RefundBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)

	LockSeats(ctx context.Context, tx *gorm.DB, showtimeID string, ids []string) ([]models.ShowtimeSeat, error)
	HoldSeat(ctx context.Context, tx *gorm.DB, id, userID string, heldUntil time.Time) (int64, error)
	ReleaseHeldSeat(ctx context.Context, tx *gorm.DB, id, userID string, version int64) (int64, error)
	SellSeat(ctx context.Context, tx *gorm.DB, id, userID string, version int64) (int64, error)
	SellSeatAtCounter(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	SeatsByIDs(ctx context.Context, tx *gorm.DB, ids []string) (map[string]models.Seat, error)
	PricesByHall(ctx context.Context, tx *gorm.DB, hallID string) (map[string]int64, error)

	CreateTickets(ctx context.Context, tx *gorm.DB, tickets []models.Ticket) error
	TicketRows(ctx context.Context, bookingID string) ([]TicketRow, error)
	ShowtimeInfos(ctx context.Context, ids []string) (map[string]ShowtimeInfoRow, error)
	TicketForGate(ctx context.Context, ref string) (*TicketGateRow, error)
	RedeemTicket(ctx context.Context, tx *gorm.DB, ticketID string) (int64, error)

	SweepExpiredHolds(ctx context.Context, limit int) ([]models.ShowtimeSeat, error)
	ExpireOverdueUnpaid(ctx context.Context, limit int) (int64, error)
	StuckPaidIDs(ctx context.Context, limit int) ([]string, error)
	DeferFinalize(ctx context.Context, id string) (int, error)

	PendingEmailIDs(ctx context.Context, limit int) ([]string, error)
	ClaimEmail(ctx context.Context, id string) (int64, error)
	MarkEmailSent(ctx context.Context, id string) error
	ReleaseEmailClaim(ctx context.Context, id string) (int, error)
	BookingHeader(ctx context.Context, id string) (*BookingHeader, error)
}

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) conn(ctx context.Context, tx *gorm.DB) *gorm.DB {
	if tx == nil {
		tx = r.db
	}
	return tx.WithContext(ctx)
}

func forUpdate() clause.Locking { return clause.Locking{Strength: "UPDATE"} }

func forShare() clause.Locking { return clause.Locking{Strength: "SHARE"} }

func firstOrNil[T any](q *gorm.DB, what string) (*T, error) {
	var row T
	if err := q.First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return &row, nil
}

// Now reads the database clock so hold/expiry checks do not depend on app server clock drift.
func (r *bookingRepository) Now(ctx context.Context, tx *gorm.DB) (time.Time, error) {
	var now time.Time
	if err := r.conn(ctx, tx).Raw("SELECT NOW()").Scan(&now).Error; err != nil {
		return time.Time{}, fmt.Errorf("read db clock: %w", err)
	}
	return now, nil
}

func (r *bookingRepository) FindByID(ctx context.Context, id string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.db.WithContext(ctx).Where("id = ?", id), "find booking")
}

func (r *bookingRepository) FindByIDAndUser(ctx context.Context, id, userID string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID), "find user booking")
}

func (r *bookingRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.Booking, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}
	var bookings []models.Booking
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&bookings).Error; err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	return bookings, total, nil
}

// LockUserShow serializes one user's holds on one showtime (tabs, retries) so each sees the
// booking the previous one committed. The two-key lock does not collide with the one-key hall lock.
func (r *bookingRepository) LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error {
	if err := r.conn(ctx, tx).Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", userID, showtimeID).Error; err != nil {
		return fmt.Errorf("lock user showtime: %w", err)
	}
	return nil
}

// UserActive share-locks the user row so an admin locking the account waits for in-flight
// holds and payments. Take it before any booking lock to keep one lock order.
func (r *bookingRepository) UserActive(ctx context.Context, tx *gorm.DB, userID string) (bool, error) {
	var active []bool
	if err := r.conn(ctx, tx).Raw(`SELECT active FROM users WHERE id = ? AND deleted_at IS NULL FOR SHARE`, userID).
		Scan(&active).Error; err != nil {
		return false, fmt.Errorf("read user active: %w", err)
	}
	return len(active) == 1 && active[0], nil
}

// Every state change of a booking (pay, IPN, confirm, refund, replace) takes this row lock.
func (r *bookingRepository) LockBooking(ctx context.Context, tx *gorm.DB, id string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.conn(ctx, tx).Clauses(forUpdate()).Where("id = ?", id), "lock booking")
}

// Any status matches: an idempotency key backs one request only.
func (r *bookingRepository) LockLatestByKey(ctx context.Context, tx *gorm.DB, key string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.conn(ctx, tx).Clauses(forUpdate()).
		Where("idempotency_key = ?", key).Order("created_at DESC"), "lock booking by idempotency key")
}

func (r *bookingRepository) LockPendingByUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.conn(ctx, tx).Clauses(forUpdate()).
		Where("user_id = ? AND showtime_id = ? AND status = ?", userID, showtimeID, models.BookingPending), "lock pending booking")
}

// LockShowtime takes a share lock: holds and confirms run in parallel, but an
// admin closing/rescheduling the show waits for them (and vice versa).
func (r *bookingRepository) LockShowtime(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error) {
	return firstOrNil[models.Showtime](r.conn(ctx, tx).Clauses(forShare()).Where("id = ?", id), "lock showtime")
}

func (r *bookingRepository) MovieShowing(ctx context.Context, tx *gorm.DB, movieID string) (bool, error) {
	var status []string
	if err := r.conn(ctx, tx).Raw(`SELECT status FROM movies WHERE id = ? AND deleted_at IS NULL`, movieID).
		Scan(&status).Error; err != nil {
		return false, fmt.Errorf("read movie status: %w", err)
	}
	return len(status) == 1 && status[0] == models.MovieStatusShowing, nil
}

func (r *bookingRepository) Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error {
	if err := r.conn(ctx, tx).Create(booking).Error; err != nil {
		return fmt.Errorf("create booking: %w", err)
	}
	return nil
}

func (r *bookingRepository) CreateBookingSeats(ctx context.Context, tx *gorm.DB, seats []models.BookingSeat) error {
	if len(seats) == 0 {
		return nil
	}
	if err := r.conn(ctx, tx).Create(&seats).Error; err != nil {
		return fmt.Errorf("create booking seats: %w", err)
	}
	return nil
}

func (r *bookingRepository) BookingSeats(ctx context.Context, tx *gorm.DB, bookingID string) ([]models.BookingSeat, error) {
	var seats []models.BookingSeat
	if err := r.conn(ctx, tx).Where("booking_id = ?", bookingID).
		Order("showtime_seat_id").Find(&seats).Error; err != nil {
		return nil, fmt.Errorf("find booking seats: %w", err)
	}
	return seats, nil
}

func (r *bookingRepository) ExpireBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NULL`,
		models.BookingExpired, reason, id, models.BookingPending)
	if res.Error != nil {
		return 0, fmt.Errorf("expire booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// Expired bookings can be paid too (late payment); they are refunded right after.
func (r *bookingRepository) SetPaid(ctx context.Context, tx *gorm.DB, id, paymentID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET paid_at = NOW(), payment_id = ?, updated_at = NOW()
		WHERE id = ? AND paid_at IS NULL AND status IN (?, ?)`,
		paymentID, id, models.BookingPending, models.BookingExpired)
	if res.Error != nil {
		return 0, fmt.Errorf("mark booking paid: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// PENDING -> CONFIRMED CAS; 0 rows means someone else already moved the booking.
func (r *bookingRepository) ConfirmBooking(ctx context.Context, tx *gorm.DB, id string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = NULL, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NOT NULL`,
		models.BookingConfirmed, id, models.BookingPending)
	if res.Error != nil {
		return 0, fmt.Errorf("confirm booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// A confirmed booking never matches, so a late refund can not undo a sale.
func (r *bookingRepository) RefundBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status IN (?, ?) AND paid_at IS NOT NULL`,
		models.BookingRefunded, reason, id, models.BookingPending, models.BookingExpired)
	if res.Error != nil {
		return 0, fmt.Errorf("refund booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// Seats are locked in id order so concurrent holds and confirms never deadlock.
func (r *bookingRepository) LockSeats(ctx context.Context, tx *gorm.DB, showtimeID string, ids []string) ([]models.ShowtimeSeat, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var seats []models.ShowtimeSeat
	if err := r.conn(ctx, tx).Clauses(forUpdate()).
		Where("showtime_id = ? AND id IN ?", showtimeID, ids).
		Order("id").Find(&seats).Error; err != nil {
		return nil, fmt.Errorf("lock showtime seats: %w", err)
	}
	return seats, nil
}

// HoldSeat returns the seat's new fencing version.
func (r *bookingRepository) HoldSeat(ctx context.Context, tx *gorm.DB, id, userID string, heldUntil time.Time) (int64, error) {
	var version int64
	res := r.conn(ctx, tx).Raw(`UPDATE showtime_seats
		SET status = ?, held_by = ?, held_until = ?, version = version + 1, updated_at = NOW()
		WHERE id = ? RETURNING version`,
		models.SeatStatusHeld, userID, heldUntil, id).Scan(&version)
	if res.Error != nil {
		return 0, fmt.Errorf("hold seat: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, fmt.Errorf("hold seat %s: row vanished", id)
	}
	return version, nil
}

// Frees the seat only while it is still the same hold (user + version); a seat swept or taken over is left alone.
func (r *bookingRepository) ReleaseHeldSeat(ctx context.Context, tx *gorm.DB, id, userID string, version int64) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE showtime_seats
		SET status = ?, held_by = NULL, held_until = NULL, version = version + 1, updated_at = NOW()
		WHERE id = ? AND status = ? AND held_by = ? AND version = ?`,
		models.SeatStatusAvailable, id, models.SeatStatusHeld, userID, version)
	if res.Error != nil {
		return 0, fmt.Errorf("release seat: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// HELD -> SOLD CAS fenced by the hold version.
func (r *bookingRepository) SellSeat(ctx context.Context, tx *gorm.DB, id, userID string, version int64) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE showtime_seats
		SET status = ?, held_by = NULL, held_until = NULL, version = version + 1, updated_at = NOW()
		WHERE id = ? AND status = ? AND held_by = ? AND version = ? AND held_until > NOW()`,
		models.SeatStatusSold, id, models.SeatStatusHeld, userID, version)
	if res.Error != nil {
		return 0, fmt.Errorf("sell seat: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *bookingRepository) SeatsByIDs(ctx context.Context, tx *gorm.DB, ids []string) (map[string]models.Seat, error) {
	byID := make(map[string]models.Seat, len(ids))
	if len(ids) == 0 {
		return byID, nil
	}
	var seats []models.Seat
	if err := r.conn(ctx, tx).Where("id IN ?", ids).Find(&seats).Error; err != nil {
		return nil, fmt.Errorf("find seats by ids: %w", err)
	}
	for _, s := range seats {
		byID[s.ID] = s
	}
	return byID, nil
}

func (r *bookingRepository) PricesByHall(ctx context.Context, tx *gorm.DB, hallID string) (map[string]int64, error) {
	var prices []models.HallPrice
	if err := r.conn(ctx, tx).Where("hall_id = ?", hallID).Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("find hall prices: %w", err)
	}
	byType := make(map[string]int64, len(prices))
	for _, p := range prices {
		byType[p.SeatType] = p.Price
	}
	return byType, nil
}

func (r *bookingRepository) CreateTickets(ctx context.Context, tx *gorm.DB, tickets []models.Ticket) error {
	if err := r.conn(ctx, tx).Create(&tickets).Error; err != nil {
		return fmt.Errorf("create tickets: %w", err)
	}
	return nil
}

func (r *bookingRepository) TicketRows(ctx context.Context, bookingID string) ([]TicketRow, error) {
	var rows []TicketRow
	if err := r.db.WithContext(ctx).Raw(`SELECT t.id, t.booking_id, t.showtime_seat_id, t.price, t.code, t.status,
			s.row_label, s.col_number, s.seat_type
		FROM tickets t
		JOIN showtime_seats ss ON ss.id = t.showtime_seat_id
		JOIN seats s ON s.id = ss.seat_id
		WHERE t.booking_id = ?
		ORDER BY s.row_index, s.col_number`, bookingID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find tickets: %w", err)
	}
	return rows, nil
}

// Deleted movies and halls are included so an old order still shows what was bought.
func (r *bookingRepository) ShowtimeInfos(ctx context.Context, ids []string) (map[string]ShowtimeInfoRow, error) {
	out := make(map[string]ShowtimeInfoRow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []ShowtimeInfoRow
	if err := r.db.WithContext(ctx).Raw(`SELECT st.id, st.movie_id, m.title AS movie_title,
			m.age_rating AS age_rating, st.hall_id, h.name AS hall_name, st.start_at, st.end_at
		FROM showtimes st
		JOIN movies m ON m.id = st.movie_id
		JOIN halls h ON h.id = st.hall_id
		WHERE st.id IN ?`, ids).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find showtime infos: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func (r *bookingRepository) TicketForGate(ctx context.Context, ref string) (*TicketGateRow, error) {
	ref = strings.TrimSpace(ref)
	where, arg := "t.code = ?", strings.ToUpper(ref)
	// uuid.Parse also accepts 32 bare hex chars (a ticket code), so only the dashed form is an id.
	if _, err := uuid.Parse(ref); err == nil && len(ref) == 36 {
		where, arg = "t.id = ?", ref
	}
	var rows []TicketGateRow
	if err := r.db.WithContext(ctx).Raw(`SELECT t.id, t.code, t.status, b.status AS booking_status,
			b.showtime_id, st.start_at, h.name AS hall_name, m.title AS movie_title,
			m.age_rating AS age_rating, s.row_label, s.col_number
		FROM tickets t
		JOIN bookings b ON b.id = t.booking_id
		JOIN showtimes st ON st.id = b.showtime_id
		JOIN halls h ON h.id = st.hall_id
		JOIN movies m ON m.id = st.movie_id
		JOIN showtime_seats ss ON ss.id = t.showtime_seat_id
		JOIN seats s ON s.id = ss.seat_id
		WHERE `+where+` LIMIT 1`, arg).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find ticket for gate: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (r *bookingRepository) RedeemTicket(ctx context.Context, tx *gorm.DB, ticketID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE tickets SET status = ?, updated_at = NOW() WHERE id = ? AND status = ?`,
		models.TicketRedeemed, ticketID, models.TicketIssued)
	if res.Error != nil {
		return 0, fmt.Errorf("redeem ticket: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// SKIP LOCKED: seats locked by an in-flight hold/confirm are left to the next tick, so the sweep
// never waits or deadlocks with them (it does not lock in id order). The outer WHERE is re-checked
// under the row lock, so a seat just sold or re-held is skipped. One audit row per showtime.
func (r *bookingRepository) SweepExpiredHolds(ctx context.Context, limit int) ([]models.ShowtimeSeat, error) {
	var seats []models.ShowtimeSeat
	if err := r.db.WithContext(ctx).Raw(`
		WITH released AS (
		UPDATE showtime_seats SET
			status = 'available',
			held_by = NULL,
			held_until = NULL,
			version = version + 1,
			updated_at = NOW()
		WHERE status = 'held'
		  AND held_until < NOW()
		  AND id IN (
			SELECT id FROM showtime_seats
			WHERE status = 'held' AND held_until < NOW()
			ORDER BY held_until
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		  )
		RETURNING id, showtime_id, status, version
		), logged AS (
			INSERT INTO audit_logs (id, actor_role, action, resource_type, resource_id, after_json, outcome, created_at)
			SELECT gen_random_uuid(), 'system', 'seats.release_expired', 'showtime', showtime_id::text,
				jsonb_build_object('seats', COUNT(*), 'reason', 'hold_expired', 'source', 'sweep'), 'success', NOW()
			FROM released
			GROUP BY showtime_id
		)
		SELECT id, showtime_id, status, version FROM released`, limit).Scan(&seats).Error; err != nil {
		return nil, fmt.Errorf("sweep expired holds: %w", err)
	}
	return seats, nil
}

// Rows locked by an in-flight payment are skipped (SKIP LOCKED) and re-checked on the next run.
func (r *bookingRepository) ExpireOverdueUnpaid(ctx context.Context, limit int) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`
		WITH expired AS (
			UPDATE bookings SET status = 'expired', status_reason = 'hold_expired', updated_at = NOW()
			WHERE status = 'pending' AND paid_at IS NULL AND expires_at < NOW()
			  AND id IN (
				SELECT id FROM bookings
				WHERE status = 'pending' AND paid_at IS NULL AND expires_at < NOW()
				ORDER BY expires_at
				LIMIT ?
				FOR UPDATE SKIP LOCKED
			  )
			RETURNING id, showtime_id
		)
		INSERT INTO audit_logs (id, actor_role, action, resource_type, resource_id, after_json, outcome, created_at)
		SELECT gen_random_uuid(), 'system', 'orders.expire', 'booking', id::text,
			jsonb_build_object('status', 'expired', 'reason', 'hold_expired', 'showtime_id', showtime_id, 'source', 'sweep'),
			'success', NOW()
		FROM expired`, limit)
	if res.Error != nil {
		return 0, fmt.Errorf("expire overdue bookings: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// Paid bookings still PENDING: overdue, or paid over a minute ago (the confirm after payment crashed).
func (r *bookingRepository) StuckPaidIDs(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status = ? AND paid_at IS NOT NULL", models.BookingPending).
		Where("expires_at < NOW() OR paid_at < NOW() - INTERVAL '1 minute'").
		Where("next_finalize_at IS NULL OR next_finalize_at <= NOW()").
		Order("next_finalize_at NULLS FIRST, paid_at").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("find stuck paid bookings: %w", err)
	}
	return ids, nil
}

// Backoff: 1 minute doubled per failure, at most 1 hour. Returns the failed tries so far.
func (r *bookingRepository) DeferFinalize(ctx context.Context, id string) (int, error) {
	var attempts int
	if err := r.db.WithContext(ctx).Raw(`UPDATE bookings
		SET next_finalize_at = NOW() + LEAST(INTERVAL '1 hour', INTERVAL '1 minute' * power(2, finalize_attempts)),
			finalize_attempts = finalize_attempts + 1
		WHERE id = ? RETURNING finalize_attempts`, id).Scan(&attempts).Error; err != nil {
		return 0, fmt.Errorf("defer finalize: %w", err)
	}
	return attempts, nil
}

// After this many tries the email is given up; the tickets stay on the web.
// idx_bookings_email_pending hardcodes the same limit.
const MaxTicketEmailAttempts = 6

// Given-up emails drop out, so they never starve newer ones.
func (r *bookingRepository) PendingEmailIDs(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status = ? AND sold_via = ? AND email_sent_at IS NULL AND email_attempts < ?",
			models.BookingConfirmed, models.SoldViaOnline, MaxTicketEmailAttempts).
		Where("email_claimed_until IS NULL OR email_claimed_until < NOW()").
		Order("created_at").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("find bookings awaiting email: %w", err)
	}
	return ids, nil
}

// The 5-minute lease keeps two workers from mailing the same booking; a worker dying mid-send
// leaves it for retry after the lease. 0 rows means nothing to do.
func (r *bookingRepository) ClaimEmail(ctx context.Context, id string) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`UPDATE bookings
		SET email_claimed_until = NOW() + INTERVAL '5 minutes', email_attempts = email_attempts + 1
		WHERE id = ? AND status = ? AND sold_via = ? AND email_sent_at IS NULL AND email_attempts < ?
		  AND (email_claimed_until IS NULL OR email_claimed_until < NOW())`,
		id, models.BookingConfirmed, models.SoldViaOnline, MaxTicketEmailAttempts)
	if res.Error != nil {
		return 0, fmt.Errorf("claim ticket email: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *bookingRepository) MarkEmailSent(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE bookings SET email_sent_at = NOW(), email_claimed_until = NULL
		WHERE id = ?`, id).Error; err != nil {
		return fmt.Errorf("mark ticket email sent: %w", err)
	}
	return nil
}

// Backoff: 1 minute doubled per try, at most 1 hour. Returns the tries made so far.
func (r *bookingRepository) ReleaseEmailClaim(ctx context.Context, id string) (int, error) {
	var attempts int
	if err := r.db.WithContext(ctx).Raw(`UPDATE bookings
		SET email_claimed_until = NOW() + LEAST(INTERVAL '1 hour', INTERVAL '1 minute' * power(2, GREATEST(email_attempts - 1, 0)))
		WHERE id = ? RETURNING email_attempts`, id).Scan(&attempts).Error; err != nil {
		return 0, fmt.Errorf("release ticket email claim: %w", err)
	}
	return attempts, nil
}

func (r *bookingRepository) BookingHeader(ctx context.Context, id string) (*BookingHeader, error) {
	var rows []BookingHeader
	if err := r.db.WithContext(ctx).Raw(`SELECT b.id, b.status, b.total_amount, b.sold_via, u.email, u.full_name,
			m.title AS movie_title, m.age_rating AS age_rating, h.name AS hall_name, st.start_at
		FROM bookings b
		LEFT JOIN users u ON u.id = b.user_id
		JOIN showtimes st ON st.id = b.showtime_id
		JOIN movies m ON m.id = st.movie_id
		JOIN halls h ON h.id = st.hall_id
		WHERE b.id = ?`, id).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find booking header: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// CreateCounterBooking inserts a walk-in sale: no account, no payment, cash
// collected at the counter (paid_at = NOW()), straight to confirmed by the service.
func (r *bookingRepository) CreateCounterBooking(ctx context.Context, tx *gorm.DB, b *models.Booking) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	if err := r.conn(ctx, tx).Exec(`INSERT INTO bookings
		(id, showtime_id, user_id, sold_via, customer_name, customer_phone, status, total_amount,
		 paid_at, created_at, updated_at)
		VALUES (?, ?, NULL, ?, NULLIF(?, ''), NULLIF(?, ''), 'pending', ?, NOW(), NOW(), NOW())`,
		b.ID, b.ShowtimeID, models.SoldViaCounter, b.CustomerName, b.CustomerPhone, b.TotalAmount).Error; err != nil {
		return fmt.Errorf("create counter booking: %w", err)
	}
	return nil
}

// SellSeatAtCounter sells a free seat directly at the till: only an AVAILABLE
// seat can be upgraded to SOLD, fencing concurrent holds and finalizes.
func (r *bookingRepository) SellSeatAtCounter(ctx context.Context, tx *gorm.DB, id string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE showtime_seats
		SET status = ?, held_by = NULL, held_until = NULL, version = version + 1, updated_at = NOW()
		WHERE id = ? AND status = ?`,
		models.SeatStatusSold, id, models.SeatStatusAvailable)
	if res.Error != nil {
		return 0, fmt.Errorf("sell seat at counter: %w", res.Error)
	}
	return res.RowsAffected, nil
}
