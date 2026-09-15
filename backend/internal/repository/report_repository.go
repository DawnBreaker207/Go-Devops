package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type ShowtimeBoardRow struct {
	ID         string    `gorm:"column:id"`
	StartAt    time.Time `gorm:"column:start_at"`
	EndAt      time.Time `gorm:"column:end_at"`
	Status     string    `gorm:"column:status"`
	MovieTitle string    `gorm:"column:movie_title"`
	HallName   string    `gorm:"column:hall_name"`
	Capacity   int       `gorm:"column:capacity"`
	Held       int       `gorm:"column:held"`
	Sold       int       `gorm:"column:sold"`
	CheckedIn  int       `gorm:"column:checked_in"`
}

type ShowtimeTicketRow struct {
	ID        string    `gorm:"column:id"`
	BookingID string    `gorm:"column:booking_id"`
	Status    string    `gorm:"column:status"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	RowLabel  string    `gorm:"column:row_label"`
	ColNumber int       `gorm:"column:col_number"`
	SeatType  string    `gorm:"column:seat_type"`
}

type ReportRepository interface {
	UpsertDailyAggregate(ctx context.Context, reportDate string, from, to time.Time) error
	DailyAggregate(ctx context.Context, reportDate string) (*models.DailyAggregate, error)
	DailyAggregates(ctx context.Context, from, to string) ([]models.DailyAggregate, error)
	ShowtimeBoard(ctx context.Context, from, to time.Time) ([]ShowtimeBoardRow, error)
	ShowtimeTickets(ctx context.Context, showtimeID, status string) ([]ShowtimeTicketRow, error)
	CounterSalesDay(ctx context.Context, from, to time.Time) (count, total int64, err error)
	LiveDayAggregate(ctx context.Context, from, to time.Time) (LiveAggregateRow, error)
}

// LiveAggregateRow is the same shape as a daily_aggregates row, computed on
// the spot for a window that closeDay may not have run for yet (typically
// "today").
type LiveAggregateRow struct {
	TotalRevenue  int64   `gorm:"column:total_revenue"`
	TicketsSold   int     `gorm:"column:tickets_sold"`
	SeatsSold     int     `gorm:"column:seats_sold"`
	Capacity      int     `gorm:"column:capacity"`
	OccupancyRate float64 `gorm:"column:occupancy_rate"`
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

// Revenue and tickets count CONFIRMED bookings paid in [from, to); seats and occupancy count
// showtimes starting in it. ON CONFLICT makes a re-run replace the day's numbers, never add to them.
const upsertDailyAggregateSQL = `
WITH paid AS (
	SELECT b.id, b.total_amount
	FROM bookings b
	WHERE b.status = 'confirmed' AND b.paid_at >= @from AND b.paid_at < @to
),
shows AS (
	SELECT st.id, st.start_at, m.title AS movie_title, h.name AS hall_name,
		COUNT(ss.id) FILTER (WHERE NOT s.is_gap) AS capacity,
		COUNT(ss.id) FILTER (WHERE ss.status = 'sold') AS seats_sold,
		(SELECT COUNT(*) FROM tickets t JOIN bookings b ON b.id = t.booking_id
			WHERE b.showtime_id = st.id AND b.status = 'confirmed' AND t.status = 'redeemed') AS checked_in,
		(SELECT COALESCE(SUM(b.total_amount), 0) FROM bookings b
			WHERE b.showtime_id = st.id AND b.status = 'confirmed') AS revenue
	FROM showtimes st
	JOIN movies m ON m.id = st.movie_id
	JOIN halls h ON h.id = st.hall_id
	LEFT JOIN showtime_seats ss ON ss.showtime_id = st.id
	LEFT JOIN seats s ON s.id = ss.seat_id
	WHERE st.deleted_at IS NULL AND st.start_at >= @from AND st.start_at < @to
	GROUP BY st.id, m.title, h.name
),
totals AS (
	SELECT COALESCE(SUM(capacity), 0) AS capacity, COALESCE(SUM(seats_sold), 0) AS seats_sold FROM shows
)
INSERT INTO daily_aggregates (id, report_date, total_revenue, tickets_sold, seats_sold, capacity,
	occupancy_rate, breakdown, created_at, updated_at)
SELECT gen_random_uuid(), CAST(@date AS date),
	(SELECT COALESCE(SUM(total_amount), 0) FROM paid),
	(SELECT COUNT(*) FROM tickets t JOIN paid ON paid.id = t.booking_id),
	totals.seats_sold,
	totals.capacity,
	CASE WHEN totals.capacity = 0 THEN 0 ELSE ROUND(100.0 * totals.seats_sold / totals.capacity, 2) END,
	jsonb_build_object('showtimes', COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'showtime_id', id, 'movie', movie_title, 'hall', hall_name, 'start_at', start_at,
			'capacity', capacity, 'seats_sold', seats_sold, 'checked_in', checked_in, 'revenue', revenue
		) ORDER BY start_at) FROM shows), '[]'::jsonb)),
	NOW(), NOW()
FROM totals
ON CONFLICT (report_date) DO UPDATE SET
	total_revenue = EXCLUDED.total_revenue,
	tickets_sold = EXCLUDED.tickets_sold,
	seats_sold = EXCLUDED.seats_sold,
	capacity = EXCLUDED.capacity,
	occupancy_rate = EXCLUDED.occupancy_rate,
	breakdown = EXCLUDED.breakdown,
	updated_at = NOW()`

func (r *reportRepository) UpsertDailyAggregate(ctx context.Context, reportDate string, from, to time.Time) error {
	if err := r.db.WithContext(ctx).Exec(upsertDailyAggregateSQL,
		map[string]any{"date": reportDate, "from": from, "to": to}).Error; err != nil {
		return fmt.Errorf("upsert daily aggregate %s: %w", reportDate, err)
	}
	return nil
}

func (r *reportRepository) DailyAggregate(ctx context.Context, reportDate string) (*models.DailyAggregate, error) {
	return firstOrNil[models.DailyAggregate](r.db.WithContext(ctx).Where("report_date = ?", reportDate), "find daily aggregate")
}

func (r *reportRepository) DailyAggregates(ctx context.Context, from, to string) ([]models.DailyAggregate, error) {
	var rows []models.DailyAggregate
	if err := r.db.WithContext(ctx).Where("report_date BETWEEN ? AND ?", from, to).
		Order("report_date").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list daily aggregates: %w", err)
	}
	return rows, nil
}

func (r *reportRepository) ShowtimeBoard(ctx context.Context, from, to time.Time) ([]ShowtimeBoardRow, error) {
	var rows []ShowtimeBoardRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT st.id, st.start_at, st.end_at, st.status, m.title AS movie_title, h.name AS hall_name,
			COUNT(ss.id) FILTER (WHERE NOT s.is_gap) AS capacity,
			COUNT(ss.id) FILTER (WHERE ss.status = 'held' AND ss.held_until > NOW()) AS held,
			COUNT(ss.id) FILTER (WHERE ss.status = 'sold') AS sold,
			(SELECT COUNT(*) FROM tickets t JOIN bookings b ON b.id = t.booking_id
				WHERE b.showtime_id = st.id AND b.status = 'confirmed' AND t.status = 'redeemed') AS checked_in
		FROM showtimes st
		JOIN movies m ON m.id = st.movie_id
		JOIN halls h ON h.id = st.hall_id
		LEFT JOIN showtime_seats ss ON ss.showtime_id = st.id
		LEFT JOIN seats s ON s.id = ss.seat_id
		WHERE st.deleted_at IS NULL AND st.start_at >= ? AND st.start_at < ?
		GROUP BY st.id, m.title, h.name
		ORDER BY st.start_at`, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("staff showtime board: %w", err)
	}
	return rows, nil
}

func (r *reportRepository) ShowtimeTickets(ctx context.Context, showtimeID, status string) ([]ShowtimeTicketRow, error) {
	q := `SELECT t.id, t.booking_id, t.status, t.updated_at, s.row_label, s.col_number, s.seat_type
		FROM tickets t
		JOIN bookings b ON b.id = t.booking_id
		JOIN showtime_seats ss ON ss.id = t.showtime_seat_id
		JOIN seats s ON s.id = ss.seat_id
		WHERE b.showtime_id = ? AND b.status = 'confirmed'`
	args := []any{showtimeID}
	if status != "" {
		q += ` AND t.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY s.row_index, s.col_number`
	var rows []ShowtimeTicketRow
	if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("showtime tickets: %w", err)
	}
	return rows, nil
}

// CounterSalesDay sums the walk-in sales collected at the counter within [from, to).
func (r *reportRepository) CounterSalesDay(ctx context.Context, from, to time.Time) (int64, int64, error) {
	var row struct {
		Count int64 `gorm:"column:count"`
		Total int64 `gorm:"column:total"`
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT COUNT(*) AS count, COALESCE(SUM(total_amount), 0) AS total
		FROM bookings
		WHERE sold_via = ? AND status != 'pending' AND paid_at >= ? AND paid_at < ?`,
		models.SoldViaCounter, from, to).Scan(&row).Error; err != nil {
		return 0, 0, fmt.Errorf("counter sales day: %w", err)
	}
	return row.Count, row.Total, nil
}

// LiveDayAggregate is the read-only twin of upsertDailyAggregateSQL's
// computation, for a window (typically "today") that hasn't been closed by
// the closeDay job yet — an admin overview should not have to wait for it.
const liveDayAggregateSQL = `
WITH paid AS (
	SELECT b.id, b.total_amount
	FROM bookings b
	WHERE b.status = 'confirmed' AND b.paid_at >= @from AND b.paid_at < @to
),
shows AS (
	SELECT st.id,
		COUNT(ss.id) FILTER (WHERE NOT s.is_gap) AS capacity,
		COUNT(ss.id) FILTER (WHERE ss.status = 'sold') AS seats_sold
	FROM showtimes st
	LEFT JOIN showtime_seats ss ON ss.showtime_id = st.id
	LEFT JOIN seats s ON s.id = ss.seat_id
	WHERE st.deleted_at IS NULL AND st.start_at >= @from AND st.start_at < @to
	GROUP BY st.id
),
totals AS (
	SELECT COALESCE(SUM(capacity), 0) AS capacity, COALESCE(SUM(seats_sold), 0) AS seats_sold FROM shows
)
SELECT
	(SELECT COALESCE(SUM(total_amount), 0) FROM paid) AS total_revenue,
	(SELECT COUNT(*) FROM tickets t JOIN paid ON paid.id = t.booking_id) AS tickets_sold,
	totals.seats_sold,
	totals.capacity,
	CASE WHEN totals.capacity = 0 THEN 0 ELSE ROUND(100.0 * totals.seats_sold / totals.capacity, 2) END AS occupancy_rate
FROM totals`

func (r *reportRepository) LiveDayAggregate(ctx context.Context, from, to time.Time) (LiveAggregateRow, error) {
	var row LiveAggregateRow
	if err := r.db.WithContext(ctx).Raw(liveDayAggregateSQL, map[string]any{"from": from, "to": to}).
		Scan(&row).Error; err != nil {
		return LiveAggregateRow{}, fmt.Errorf("live day aggregate: %w", err)
	}
	return row, nil
}
