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

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
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
	BookingID     string    `gorm:"column:booking_id"`
	BookingStatus string    `gorm:"column:booking_status"`
	ShowtimeID    string    `gorm:"column:showtime_id"`
	StartAt       time.Time `gorm:"column:start_at"`
	HallName      string    `gorm:"column:hall_name"`
	MovieTitle    string    `gorm:"column:movie_title"`
	AgeRating     string    `gorm:"column:age_rating"`
	RowLabel      string    `gorm:"column:row_label"`
	ColNumber     int       `gorm:"column:col_number"`
}

// TicketOwnerRow: ticket plus owner identity for GET /tickets/{id}/qr authorization.
type TicketOwnerRow struct {
	ID        string `gorm:"column:id"`
	Code      string `gorm:"column:code"`
	Status    string `gorm:"column:status"`
	BookingID string `gorm:"column:booking_id"`
	UserID    string `gorm:"column:user_id"`
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

// AdminOrderRow: one operator-list booking (account, showtime, carried payment attempt).
// Nullable text columns are plain strings, like BookingHeader.
type AdminOrderRow struct {
	ID           string `gorm:"column:id"`
	UserID       string `gorm:"column:user_id"`
	ShowtimeID   string `gorm:"column:showtime_id"`
	Status       string `gorm:"column:status"`
	StatusReason string `gorm:"column:status_reason"`
	TotalAmount  int64  `gorm:"column:total_amount"`
	// Undiscounted subtotal above; what the customer owes is TotalAmount-DiscountAmount.
	DiscountAmount int64      `gorm:"column:discount_amount"`
	SoldVia        string     `gorm:"column:sold_via"`
	CustomerName   string     `gorm:"column:customer_name"`
	CustomerPhone  string     `gorm:"column:customer_phone"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	PaidAt         *time.Time `gorm:"column:paid_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	Seats          int        `gorm:"column:seats"`

	Email    string `gorm:"column:email"`
	FullName string `gorm:"column:full_name"`
	Phone    string `gorm:"column:phone"`

	MovieID    string    `gorm:"column:movie_id"`
	MovieTitle string    `gorm:"column:movie_title"`
	AgeRating  string    `gorm:"column:age_rating"`
	HallID     string    `gorm:"column:hall_id"`
	HallName   string    `gorm:"column:hall_name"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`

	PaymentID           string     `gorm:"column:payment_id"`
	PaymentProvider     string     `gorm:"column:payment_provider"`
	PaymentTxnRef       string     `gorm:"column:payment_txn_ref"`
	PaymentStatus       string     `gorm:"column:payment_status"`
	PaymentStatusReason string     `gorm:"column:payment_status_reason"`
	PaymentAmount       int64      `gorm:"column:payment_amount"`
	PaymentPaidAmount   *int64     `gorm:"column:payment_paid_amount"`
	PaymentPaidAt       *time.Time `gorm:"column:payment_paid_at"`
	PaymentRefundedAt   *time.Time `gorm:"column:payment_refunded_at"`
}

// TransactionRow: one payment attempt for GET /users/me/transactions, with booking/movie/hall/start
// context but not the full order shape GET /orders covers.
type TransactionRow struct {
	PaymentID        string     `gorm:"column:payment_id"`
	Provider         string     `gorm:"column:provider"`
	TxnRef           string     `gorm:"column:txn_ref"`
	Status           string     `gorm:"column:status"`
	StatusReason     string     `gorm:"column:status_reason"`
	Amount           int64      `gorm:"column:amount"`
	PaidAmount       *int64     `gorm:"column:paid_amount"`
	PaidAt           *time.Time `gorm:"column:paid_at"`
	RefundedAt       *time.Time `gorm:"column:refunded_at"`
	PaymentCreatedAt time.Time  `gorm:"column:payment_created_at"`
	BookingID        string     `gorm:"column:booking_id"`
	ShowtimeID       string     `gorm:"column:showtime_id"`
	MovieTitle       string     `gorm:"column:movie_title"`
	HallName         string     `gorm:"column:hall_name"`
	StartAt          time.Time  `gorm:"column:start_at"`
}

// Methods taking tx run in the caller's transaction; a nil tx uses the plain connection.
type BookingRepository interface {
	Now(ctx context.Context, tx *gorm.DB) (time.Time, error)

	FindByID(ctx context.Context, id string) (*models.Booking, error)
	FindByIDAndUser(ctx context.Context, id, userID string) (*models.Booking, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]models.Booking, int64, error)
	AdminOrderList(ctx context.Context, query dto.AdminOrderListQuery, from, to time.Time) ([]AdminOrderRow, int64, error)
	// TransactionsByUser: every payment attempt on the user's bookings, newest first (financial view).
	TransactionsByUser(ctx context.Context, userID string, page, pageSize int) ([]TransactionRow, int64, error)

	LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error
	UserActive(ctx context.Context, tx *gorm.DB, userID string) (bool, error)
	LockBooking(ctx context.Context, tx *gorm.DB, id string) (*models.Booking, error)
	LockLatestByKey(ctx context.Context, tx *gorm.DB, idempotencyKey string) (*models.Booking, error)
	LockPendingByUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) (*models.Booking, error)
	LockShowtime(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error)
	// LockShowtimeExclusive: LockShowtime's exclusive counterpart for cancel-showtime; waits out
	// in-flight holds/pays/confirms (same discipline as ShowtimeRepository.LockForUpdate).
	LockShowtimeExclusive(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error)
	MovieShowing(ctx context.Context, tx *gorm.DB, movieID string) (bool, error)

	Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
	CreateCounterBooking(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
	CreateBookingSeats(ctx context.Context, tx *gorm.DB, seats []models.BookingSeat) error
	BookingSeats(ctx context.Context, tx *gorm.DB, bookingID string) ([]models.BookingSeat, error)

	ExpireBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	// ExtendBookingExpiry moves a pending unpaid booking's expiry; concurrent changes make it a no-op.
	ExtendBookingExpiry(ctx context.Context, tx *gorm.DB, id string, expiresAt time.Time) (int64, error)
	// SetDiscount writes (or clears, with a nil codeID and 0) the discount on a
	// booking. The WHERE pins status=pending AND paid_at IS NULL, so a discount
	// can never be attached to an order whose money has already moved; 0 rows
	// means the order changed underneath and the caller must refuse.
	SetDiscount(ctx context.Context, tx *gorm.DB, id string, codeID *string, amount int64) (int64, error)
	SetPaid(ctx context.Context, tx *gorm.DB, id, paymentID string) (int64, error)
	ConfirmBooking(ctx context.Context, tx *gorm.DB, id string) (int64, error)
	RefundBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	// RefundConfirmedBooking: RefundBooking for CONFIRMED (tickets issued), only for showtime-cancel.
	RefundConfirmedBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error)
	// VoidTicketsForBooking marks a booking's issued tickets void, for showtime-cancel.
	VoidTicketsForBooking(ctx context.Context, tx *gorm.DB, bookingID string) (int64, error)
	// BookingIDsForShowtime lists a showtime's bookings in given statuses, for the cancel cascade.
	BookingIDsForShowtime(ctx context.Context, tx *gorm.DB, showtimeID string, statuses []string) ([]string, error)
	// CancelShowtimeStatus moves a showtime to 'cancelled'; 0 rows means already cancelled.
	CancelShowtimeStatus(ctx context.Context, tx *gorm.DB, showtimeID string) (int64, error)

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
	// TicketByID looks a ticket up by id (not code) for QR; works whether or not the showtime started.
	TicketByID(ctx context.Context, id string) (*TicketOwnerRow, error)
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
	GivenUpEmails(ctx context.Context, limit int) ([]models.Booking, error)
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

// LockUserShow serializes one user's holds on one showtime; two-key lock avoids the hall lock.
func (r *bookingRepository) LockUserShow(ctx context.Context, tx *gorm.DB, userID, showtimeID string) error {
	if err := r.conn(ctx, tx).Exec("SELECT pg_advisory_xact_lock(hashtext(?), hashtext(?))", userID, showtimeID).Error; err != nil {
		return fmt.Errorf("lock user showtime: %w", err)
	}
	return nil
}

// UserActive share-locks the user row; take before any booking lock to keep one lock order.
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

// LockShowtime share lock: holds/confirms run parallel, but showtime close/reschedule waits.
func (r *bookingRepository) LockShowtime(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error) {
	return firstOrNil[models.Showtime](r.conn(ctx, tx).Clauses(forShare()).Where("id = ?", id), "lock showtime")
}

func (r *bookingRepository) LockShowtimeExclusive(ctx context.Context, tx *gorm.DB, id string) (*models.Showtime, error) {
	return firstOrNil[models.Showtime](r.conn(ctx, tx).Clauses(forUpdate()).Where("id = ?", id), "lock showtime exclusive")
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

func (r *bookingRepository) ExtendBookingExpiry(ctx context.Context, tx *gorm.DB, id string, expiresAt time.Time) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET expires_at = ?, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NULL`,
		expiresAt, id, models.BookingPending)
	if res.Error != nil {
		return 0, fmt.Errorf("extend booking expiry: %w", res.Error)
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

// Only reachable from the showtime-cancel cascade; skips RefundBooking's pending/expired path.
func (r *bookingRepository) RefundConfirmedBooking(ctx context.Context, tx *gorm.DB, id, reason string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE bookings SET status = ?, status_reason = ?, updated_at = NOW()
		WHERE id = ? AND status = ? AND paid_at IS NOT NULL`,
		models.BookingRefunded, reason, id, models.BookingConfirmed)
	if res.Error != nil {
		return 0, fmt.Errorf("refund confirmed booking: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *bookingRepository) VoidTicketsForBooking(ctx context.Context, tx *gorm.DB, bookingID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE tickets SET status = ?, updated_at = NOW() WHERE booking_id = ? AND status = ?`,
		models.TicketVoid, bookingID, models.TicketIssued)
	if res.Error != nil {
		return 0, fmt.Errorf("void tickets: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *bookingRepository) SetDiscount(ctx context.Context, tx *gorm.DB, id string, codeID *string, amount int64) (int64, error) {
	// paid_at IS NULL as well as status=pending: an order can be paid for a
	// moment before it flips to confirmed, and re-pricing it in that window would
	// change what the customer owes after they already paid.
	res := r.conn(ctx, tx).Model(&models.Booking{}).
		Where("id = ? AND status = ? AND paid_at IS NULL", id, models.BookingPending).
		Updates(map[string]any{
			"discount_code_id": codeID,
			"discount_amount":  amount,
			"updated_at":       time.Now(),
		})
	if res.Error != nil {
		return 0, fmt.Errorf("set booking discount: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *bookingRepository) BookingIDsForShowtime(ctx context.Context, tx *gorm.DB, showtimeID string, statuses []string) ([]string, error) {
	var ids []string
	if err := r.conn(ctx, tx).Model(&models.Booking{}).
		Where("showtime_id = ? AND status IN ?", showtimeID, statuses).
		Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("list bookings for showtime: %w", err)
	}
	return ids, nil
}

func (r *bookingRepository) CancelShowtimeStatus(ctx context.Context, tx *gorm.DB, showtimeID string) (int64, error) {
	res := r.conn(ctx, tx).Exec(`UPDATE showtimes SET status = ?, updated_at = NOW() WHERE id = ? AND status <> ?`,
		models.ShowtimeCancelled, showtimeID, models.ShowtimeCancelled)
	if res.Error != nil {
		return 0, fmt.Errorf("cancel showtime: %w", res.Error)
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
	if err := r.db.WithContext(ctx).Raw(`SELECT t.id, t.code, t.status, b.id AS booking_id, b.status AS booking_status,
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

func (r *bookingRepository) TicketByID(ctx context.Context, id string) (*TicketOwnerRow, error) {
	var rows []TicketOwnerRow
	if err := r.db.WithContext(ctx).Raw(`SELECT t.id, t.code, t.status, b.id AS booking_id, COALESCE(b.user_id::text, '') AS user_id
		FROM tickets t JOIN bookings b ON b.id = t.booking_id WHERE t.id = ?`, id).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("find ticket by id: %w", err)
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

// SKIP LOCKED so the sweep never waits/deadlocks with in-flight holds (no id-order locking);
// re-checked under lock, so just-sold/re-held seats are skipped. One audit row per showtime.
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
		INSERT INTO audit_logs (id, actor_role, action, resource_type, resource_id, booking_id, after_json, outcome, created_at)
		SELECT gen_random_uuid(), 'system', 'orders.expire', 'booking', id::text, id,
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

// After this many tries the email is given up (tickets stay on the web); idx_bookings_email_pending hardcodes it.
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

// GivenUpEmails: confirmed bookings whose ticket email exhausted retries; only the admin overview surfaces them.
func (r *bookingRepository) GivenUpEmails(ctx context.Context, limit int) ([]models.Booking, error) {
	var out []models.Booking
	if err := r.db.WithContext(ctx).
		Where("status = ? AND sold_via = ? AND email_sent_at IS NULL AND email_attempts >= ?",
			models.BookingConfirmed, models.SoldViaOnline, MaxTicketEmailAttempts).
		Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, fmt.Errorf("find given-up ticket emails: %w", err)
	}
	return out, nil
}

// 5-minute lease so two workers never mail the same booking; 0 rows means nothing to do.
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

// CreateCounterBooking: walk-in sale, no account/payment, cash at counter (paid_at = NOW()).
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

// SellSeatAtCounter: only AVAILABLE -> SOLD, fencing concurrent holds/finalizes.
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

// adminOrderSortColumns whitelists operator-list sort keys; values are literals, never user input.
var adminOrderSortColumns = map[string]string{
	"created_at":   "bookings.created_at",
	"paid_at":      "bookings.paid_at",
	"total_amount": "bookings.total_amount",
	"start_at":     "showtimes.start_at",
}

func adminOrderOrder(sort, order string) string {
	column := adminOrderSortColumns[sort]
	if column == "" {
		column = "bookings.created_at"
	}
	direction := "DESC"
	if strings.EqualFold(order, "asc") {
		direction = "ASC"
	}
	return column + " " + direction
}

// AdminOrderList: unscoped ListByUser with account/showtime/payment in one round trip; from/to are
// absolute instants (timezone is the service's; zero = open). Deleted catalog rows join in so old
// orders still show what was bought; users is LEFT JOIN (counter has no account, erasure scrubs in place).
func (r *bookingRepository) AdminOrderList(ctx context.Context, query dto.AdminOrderListQuery, from, to time.Time) ([]AdminOrderRow, int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&models.Booking{}).
		Joins("JOIN showtimes ON showtimes.id = bookings.showtime_id").
		Joins("JOIN movies ON movies.id = showtimes.movie_id").
		Joins("JOIN halls ON halls.id = showtimes.hall_id").
		Joins("LEFT JOIN users ON users.id = bookings.user_id").
		Joins("LEFT JOIN payments ON payments.id = bookings.payment_id")

	if query.Status != "" {
		tx = tx.Where("bookings.status = ?", query.Status)
	}
	if query.PaymentStatus != "" {
		tx = tx.Where("payments.status = ?", query.PaymentStatus)
	}
	if query.SoldVia != "" {
		tx = tx.Where("bookings.sold_via = ?", query.SoldVia)
	}
	if query.ShowtimeID != "" {
		tx = tx.Where("bookings.showtime_id = ?", query.ShowtimeID)
	}
	if query.MovieID != "" {
		tx = tx.Where("showtimes.movie_id = ?", query.MovieID)
	}
	if query.UserID != "" {
		tx = tx.Where("bookings.user_id = ?", query.UserID)
	}
	if !from.IsZero() {
		tx = tx.Where("bookings.created_at >= ?", from)
	}
	if !to.IsZero() {
		tx = tx.Where("bookings.created_at < ?", to)
	}
	if search := strings.TrimSpace(query.Search); search != "" {
		lowered := strings.ToLower(search)
		pattern := "%" + lowered + "%"
		tx = tx.Where(`bookings.id::text = ?
				OR LOWER(users.email) LIKE ? OR LOWER(users.full_name) LIKE ?
				OR LOWER(bookings.customer_name) LIKE ? OR bookings.customer_phone LIKE ?
				OR LOWER(movies.title) LIKE ?`,
			lowered, pattern, pattern, pattern, "%"+search+"%", pattern)
	}

	// Count before Select so the seats subquery stays out of count(...). Same as movieRepository.List.
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count admin orders: %w", err)
	}

	rows := make([]AdminOrderRow, 0, query.PageSize)
	if err := tx.
		Select(`bookings.id, bookings.user_id, bookings.showtime_id, bookings.status,
			bookings.status_reason, bookings.total_amount, bookings.discount_amount, bookings.sold_via,
			bookings.customer_name, bookings.customer_phone, bookings.expires_at,
			bookings.paid_at, bookings.created_at,
			(SELECT COUNT(*) FROM booking_seats bs WHERE bs.booking_id = bookings.id) AS seats,
			users.email, users.full_name, users.phone,
			showtimes.movie_id, movies.title AS movie_title, movies.age_rating AS age_rating,
			showtimes.hall_id, halls.name AS hall_name, showtimes.start_at, showtimes.end_at,
			payments.id AS payment_id, payments.provider AS payment_provider,
			payments.txn_ref AS payment_txn_ref, payments.status AS payment_status,
			payments.status_reason AS payment_status_reason, payments.amount AS payment_amount,
			payments.paid_amount AS payment_paid_amount, payments.paid_at AS payment_paid_at,
			payments.refunded_at AS payment_refunded_at`).
		Order(adminOrderOrder(query.Sort, query.Order)).
		Order("bookings.id").
		Limit(query.PageSize).
		Offset(query.Offset()).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list admin orders: %w", err)
	}

	return rows, total, nil
}

// TransactionsByUser: every attempt on the caller's bookings, newest first — incl. failed/superseded
// tries the booking no longer carries via bookings.payment_id.
func (r *bookingRepository) TransactionsByUser(ctx context.Context, userID string, page, pageSize int) ([]TransactionRow, int64, error) {
	tx := r.db.WithContext(ctx).
		Model(&models.Payment{}).
		Joins("JOIN bookings ON bookings.id = payments.booking_id").
		Joins("JOIN showtimes ON showtimes.id = bookings.showtime_id").
		Joins("JOIN movies ON movies.id = showtimes.movie_id").
		Joins("JOIN halls ON halls.id = showtimes.hall_id").
		Where("bookings.user_id = ?", userID)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	rows := make([]TransactionRow, 0, pageSize)
	if err := tx.
		Select(`payments.id AS payment_id, payments.provider, payments.txn_ref, payments.status,
			payments.status_reason, payments.amount, payments.paid_amount, payments.paid_at,
			payments.refunded_at, payments.created_at AS payment_created_at,
			bookings.id AS booking_id, bookings.showtime_id,
			movies.title AS movie_title, halls.name AS hall_name, showtimes.start_at`).
		Order("payments.created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list transactions: %w", err)
	}
	return rows, total, nil
}
