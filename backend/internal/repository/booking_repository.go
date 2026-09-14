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

// TicketRow is a ticket joined with its physical seat.
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

// TicketGateRow is what the gate needs to judge a scanned ticket.
type TicketGateRow struct {
	ID            string    `gorm:"column:id"`
	Code          string    `gorm:"column:code"`
	Status        string    `gorm:"column:status"`
	BookingStatus string    `gorm:"column:booking_status"`
	ShowtimeID    string    `gorm:"column:showtime_id"`
	StartAt       time.Time `gorm:"column:start_at"`
	HallName      string    `gorm:"column:hall_name"`
	MovieTitle    string    `gorm:"column:movie_title"`
	RowLabel      string    `gorm:"column:row_label"`
	ColNumber     int       `gorm:"column:col_number"`
}

// BookingHeader is a booking joined with its customer, movie and hall (email).
type BookingHeader struct {
	ID          string    `gorm:"column:id"`
	Status      string    `gorm:"column:status"`
	TotalAmount int64     `gorm:"column:total_amount"`
	Email       string    `gorm:"column:email"`
	FullName    string    `gorm:"column:full_name"`
	MovieTitle  string    `gorm:"column:movie_title"`
	HallName    string    `gorm:"column:hall_name"`
	StartAt     time.Time `gorm:"column:start_at"`
}

// ShowtimeInfoRow is what an order shows about its showtime.
type ShowtimeInfoRow struct {
	ID         string    `gorm:"column:id"`
	MovieID    string    `gorm:"column:movie_id"`
	MovieTitle string    `gorm:"column:movie_title"`
	HallID     string    `gorm:"column:hall_id"`
	HallName   string    `gorm:"column:hall_name"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
}

// BookingRepository owns seats-of-a-show locking, bookings and tickets.
// Methods taking tx must run inside the caller's transaction; a nil tx falls
// back to the plain connection.
type BookingRepository interface {
	Now(ctx context.Context, tx *gorm.DB) (time.Time, error)

	FindByID(ctx context.Context, id string) (*models.Booking, error)
	FindByIDAndUser(ctx context.Context, id, userID string) (*models.Booking, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.Booking, int64, error)

	LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error
	UserActive(ctx context.Context, tx *gorm.DB, userID string) (bool, error)
	LockBooking(ctx context.Context, tx *gorm.DB, id string) (*models.Booking, error)
	LockPendingByKey(ctx context.Context, tx *gorm.DB, idempotencyKey string) (*models.Booking, error)
	LockPendingByUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) (*models.Booking, error)
	LockShowtime(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error)

	Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
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

	PendingEmailIDs(ctx context.Context, limit int) ([]string, error)
	ClaimEmail(ctx context.Context, id string) (int64, error)
	ReleaseEmailClaim(ctx context.Context, id string) error
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

// firstOrNil maps "no row" to (nil, nil).
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

// Now reads the database clock: every hold/expiry comparison uses one
// authoritative time source regardless of app machine drift (R-HO6).
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

// ListByUser returns a user's bookings (page 1-based), newest first. Expired
// and refunded bookings stay in the history (E-O2).
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

// LockUserShow serializes the holds of one user on one showtime (several tabs,
// retries with the same idempotency key) so each one sees the booking the
// previous one committed, instead of racing it. Two-key advisory locks live in
// a different key space than the single-key hall lock.
func (r *bookingRepository) LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error {
	if err := r.conn(ctx, tx).Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", userID, showtimeID).Error; err != nil {
		return fmt.Errorf("lock user showtime: %w", err)
	}
	return nil
}

// UserActive reports whether the account may buy (F18: locked accounts can
// not hold or pay). The row is share-locked, so an admin locking the account
// waits for holds and payments in flight and later ones see the lock. Callers
// take it before any booking lock (see Pay) to keep one lock order.
func (r *bookingRepository) UserActive(ctx context.Context, tx *gorm.DB, userID string) (bool, error) {
	var active []bool
	if err := r.conn(ctx, tx).Raw(`SELECT active FROM users WHERE id = ? AND deleted_at IS NULL FOR SHARE`, userID).
		Scan(&active).Error; err != nil {
		return false, fmt.Errorf("read user active: %w", err)
	}
	return len(active) == 1 && active[0], nil
}

// LockBooking serializes every state change of one booking (pay, IPN,
// confirm, refund, replace) behind its row lock.
func (r *bookingRepository) LockBooking(ctx context.Context, tx *gorm.DB, id string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.conn(ctx, tx).Clauses(forUpdate()).Where("id = ?", id), "lock booking")
}

func (r *bookingRepository) LockPendingByKey(ctx context.Context, tx *gorm.DB, key string) (*models.Booking, error) {
	return firstOrNil[models.Booking](r.conn(ctx, tx).Clauses(forUpdate()).
		Where("idempotency_key = ? AND status = ?", key, models.BookingPending), "lock booking by idempotency key")
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

// ExpireBooking moves an unpaid PENDING booking to EXPIRED.
func (r *bookingRepository) ExpireBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NULL`,
		models.BookingExpired, reason, id, models.BookingPending)
	if res.Error != nil {
		return 0, fmt.Errorf("expire booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// SetPaid records that a payment attempt's money now belongs to the booking.
// Expired bookings can be paid too (late payment) — they are refunded right after.
func (r *bookingRepository) SetPaid(ctx context.Context, tx *gorm.DB, id, paymentID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET paid_at = NOW(), payment_id = ?, updated_at = NOW()
		WHERE id = ? AND paid_at IS NULL AND status IN (?, ?)`,
		paymentID, id, models.BookingPending, models.BookingExpired)
	if res.Error != nil {
		return 0, fmt.Errorf("mark booking paid: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// ConfirmBooking is the PENDING -> CONFIRMED CAS; 0 rows means someone else
// already moved the booking.
func (r *bookingRepository) ConfirmBooking(ctx context.Context, tx *gorm.DB, id string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = NULL, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NOT NULL`,
		models.BookingConfirmed, id, models.BookingPending)
	if res.Error != nil {
		return 0, fmt.Errorf("confirm booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// RefundBooking moves a PAID booking that could not be confirmed to REFUNDED.
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

// LockSeats locks showtime seats in id order, so concurrent holds and
// confirms never deadlock on each other (R-HO2).
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

// HoldSeat marks a locked seat HELD and returns its new fencing version.
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

// ReleaseHeldSeat frees a seat only while it is still held by the same hold
// (user + version); a seat swept or taken over is left alone.
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

// SellSeat is the HELD -> SOLD CAS fenced by the hold version (R-C1).
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

// SeatsByIDs maps physical seat ids to their seat (type, label, gap).
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
		ORDER BY s.row_label, s.col_number`, bookingID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find tickets: %w", err)
	}
	return rows, nil
}

// ShowtimeInfos loads movie, hall and times of showtimes in one query
// (deleted movies/halls included: an old order still shows what was bought).
func (r *bookingRepository) ShowtimeInfos(ctx context.Context, ids []string) (map[string]ShowtimeInfoRow, error) {
	out := make(map[string]ShowtimeInfoRow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []ShowtimeInfoRow
	if err := r.db.WithContext(ctx).Raw(`SELECT st.id, st.movie_id, m.title AS movie_title,
			st.hall_id, h.name AS hall_name, st.start_at, st.end_at
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

// TicketForGate resolves a scanned reference: a ticket id or its QR code.
func (r *bookingRepository) TicketForGate(ctx context.Context, ref string) (*TicketGateRow, error) {
	ref = strings.TrimSpace(ref)
	where, arg := "t.code = ?", strings.ToUpper(ref)
	// uuid.Parse also accepts 32 bare hex chars — exactly a ticket code — so
	// only the dashed form counts as an id.
	if _, err := uuid.Parse(ref); err == nil && len(ref) == 36 {
		where, arg = "t.id = ?", ref
	}
	var rows []TicketGateRow
	if err := r.db.WithContext(ctx).Raw(`SELECT t.id, t.code, t.status, b.status AS booking_status,
			b.showtime_id, st.start_at, h.name AS hall_name, m.title AS movie_title,
			s.row_label, s.col_number
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

// RedeemTicket flips ISSUED -> REDEEMED once; 0 rows means already used.
func (r *bookingRepository) RedeemTicket(ctx context.Context, tx *gorm.DB, ticketID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE tickets SET status = ?, updated_at = NOW() WHERE id = ? AND status = ?`,
		models.TicketRedeemed, ticketID, models.TicketIssued)
	if res.Error != nil {
		return 0, fmt.Errorf("redeem ticket: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// SweepExpiredHolds flips up to limit HELD seats whose hold expired (by DB
// clock) back to AVAILABLE and returns them for broadcasting. Seats locked by
// an in-flight hold/confirm are skipped (SKIP LOCKED), so the sweep never waits
// and can not deadlock with them (they lock in id order, the sweep would not);
// skipped seats are picked up by the next tick. The outer WHERE is re-checked
// under the row lock, so a seat a confirm just sold or a hold just re-took is
// skipped (E-R1). One audit row per showtime is written by the same statement
// (F13: grouped per business event, not one row per seat).
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

// ExpireOverdueUnpaid expires PENDING bookings past their hold that were never
// paid. Rows locked by an in-flight payment are skipped and re-checked. Each
// expired booking gets its audit row in the same statement (T39).
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

// StuckPaidIDs lists paid bookings still PENDING: overdue, or paid more than
// a minute ago (the confirm after payment crashed). They must be finalized.
func (r *bookingRepository) StuckPaidIDs(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status = ? AND paid_at IS NOT NULL", models.BookingPending).
		Where("expires_at < NOW() OR paid_at < NOW() - INTERVAL '1 minute'").
		Order("paid_at").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("find stuck paid bookings: %w", err)
	}
	return ids, nil
}

func (r *bookingRepository) PendingEmailIDs(ctx context.Context, limit int) ([]string, error) {
	var ids []string
	if err := r.db.WithContext(ctx).Model(&models.Booking{}).
		Where("status = ? AND email_sent_at IS NULL", models.BookingConfirmed).
		Order("created_at").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("find bookings awaiting email: %w", err)
	}
	return ids, nil
}

// ClaimEmail marks a confirmed booking's email as sent before sending, so two
// workers never mail the same booking (E-ML2). 0 rows = already claimed.
func (r *bookingRepository) ClaimEmail(ctx context.Context, id string) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`UPDATE bookings SET email_sent_at = NOW()
		WHERE id = ? AND status = ? AND email_sent_at IS NULL`, id, models.BookingConfirmed)
	if res.Error != nil {
		return 0, fmt.Errorf("claim ticket email: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// ReleaseEmailClaim undoes a claim after a failed send so a later run retries.
func (r *bookingRepository) ReleaseEmailClaim(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Exec(`UPDATE bookings SET email_sent_at = NULL WHERE id = ?`, id).Error; err != nil {
		return fmt.Errorf("release ticket email claim: %w", err)
	}
	return nil
}

func (r *bookingRepository) BookingHeader(ctx context.Context, id string) (*BookingHeader, error) {
	var rows []BookingHeader
	if err := r.db.WithContext(ctx).Raw(`SELECT b.id, b.status, b.total_amount, u.email, u.full_name,
			m.title AS movie_title, h.name AS hall_name, st.start_at
		FROM bookings b
		JOIN users u ON u.id = b.user_id
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
