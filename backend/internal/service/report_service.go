package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// ReportService works in local days of the configured timezone.
type ReportService interface {
	CloseDay(ctx context.Context, day time.Time) (*dto.DailyAggregateResponse, error)
	DailyReport(ctx context.Context, from, to string) (*dto.DailyReportResponse, error)
	StaffBoard(ctx context.Context, date string) (*dto.StaffBoardResponse, error)
	ShowtimeTickets(ctx context.Context, showtimeID, status string) ([]dto.StaffTicketResponse, error)
	BoxOfficeDay(ctx context.Context, date string) (*dto.BoxOfficeDayResponse, error)
	AdminOverview(ctx context.Context) (*dto.AdminOverviewResponse, error)
	StaffOverview(ctx context.Context, date string) (*dto.StaffOverviewResponse, error)
	AdminStats(ctx context.Context) (*dto.AdminStatsResponse, error)
}

type reportService struct {
	repo      repository.ReportRepository
	showtimes *repository.ShowtimeRepository
	payments  repository.PaymentRepository
	batchJobs repository.BatchJobRepository
	bookings  repository.BookingRepository
	location  *time.Location
}

func NewReportService(repo repository.ReportRepository, showtimes *repository.ShowtimeRepository,
	payments repository.PaymentRepository, batchJobs repository.BatchJobRepository, bookings repository.BookingRepository,
	location *time.Location) ReportService {
	return &reportService{repo: repo, showtimes: showtimes, payments: payments, batchJobs: batchJobs,
		bookings: bookings, location: location}
}

// alertListLimit bounds each operational-alert list on the admin overview.
const alertListLimit = 20

// dayBounds returns the local date and its [from, to) instants.
func (s *reportService) dayBounds(day time.Time) (string, time.Time, time.Time) {
	d := day.In(s.location)
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, s.location)
	return from.Format(dto.DateLayout), from, from.AddDate(0, 0, 1)
}

// CloseDay upserts the daily_aggregates row of day's local date; rerunning it replaces the numbers.
func (s *reportService) CloseDay(ctx context.Context, day time.Time) (*dto.DailyAggregateResponse, error) {
	date, from, to := s.dayBounds(day)
	if err := s.repo.UpsertDailyAggregate(ctx, date, from, to); err != nil {
		return nil, err
	}
	agg, err := s.repo.DailyAggregate(ctx, date)
	if err != nil {
		return nil, err
	}
	if agg == nil {
		return nil, apperrors.Internal("daily aggregate not written")
	}
	return newDailyAggregateResponse(agg), nil
}

// DailyReport returns what closeDay last wrote for [from, to] (default: the last 7 days);
// run closeDay to refresh today.
func (s *reportService) DailyReport(ctx context.Context, from, to string) (*dto.DailyReportResponse, error) {
	now := time.Now().In(s.location)
	toDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.location)
	if to != "" {
		parsed, err := time.ParseInLocation(dto.DateLayout, to, s.location)
		if err != nil {
			return nil, apperrors.Validation("to must follow format YYYY-MM-DD")
		}
		toDay = parsed
	}
	fromDay := toDay.AddDate(0, 0, -6)
	if from != "" {
		parsed, err := time.ParseInLocation(dto.DateLayout, from, s.location)
		if err != nil {
			return nil, apperrors.Validation("from must follow format YYYY-MM-DD")
		}
		fromDay = parsed
	}
	if fromDay.After(toDay) {
		return nil, apperrors.Validation("from must not be after to")
	}
	if toDay.Sub(fromDay) > 366*24*time.Hour {
		return nil, apperrors.Validation("the range can not exceed 366 days")
	}

	rows, err := s.repo.DailyAggregates(ctx, fromDay.Format(dto.DateLayout), toDay.Format(dto.DateLayout))
	if err != nil {
		return nil, err
	}
	res := &dto.DailyReportResponse{
		From: fromDay.Format(dto.DateLayout),
		To:   toDay.Format(dto.DateLayout),
		Days: make([]dto.DailyAggregateResponse, 0, len(rows)),
	}
	for i := range rows {
		day := newDailyAggregateResponse(&rows[i])
		res.TotalRevenue += day.TotalRevenue
		res.TicketsSold += day.TicketsSold
		res.Days = append(res.Days, *day)
	}
	return res, nil
}

func (s *reportService) StaffBoard(ctx context.Context, date string) (*dto.StaffBoardResponse, error) {
	day := time.Now()
	if date != "" {
		parsed, err := time.ParseInLocation(dto.DateLayout, date, s.location)
		if err != nil {
			return nil, apperrors.Validation("date must follow format YYYY-MM-DD")
		}
		day = parsed
	}
	label, from, to := s.dayBounds(day)
	rows, err := s.repo.ShowtimeBoard(ctx, from, to)
	if err != nil {
		return nil, err
	}
	res := &dto.StaffBoardResponse{Date: label, Showtimes: make([]dto.StaffShowtimeResponse, 0, len(rows))}
	for _, r := range rows {
		res.Showtimes = append(res.Showtimes, dto.StaffShowtimeResponse{
			ID:         r.ID,
			MovieTitle: r.MovieTitle,
			HallName:   r.HallName,
			StartAt:    r.StartAt,
			EndAt:      r.EndAt,
			Status:     r.Status,
			Capacity:   r.Capacity,
			Held:       r.Held,
			Sold:       r.Sold,
			Available:  r.Capacity - r.Held - r.Sold,
			CheckedIn:  r.CheckedIn,
		})
	}
	return res, nil
}

func (s *reportService) ShowtimeTickets(ctx context.Context, showtimeID, status string) ([]dto.StaffTicketResponse, error) {
	switch status {
	case "", models.TicketIssued, models.TicketRedeemed:
	default:
		return nil, apperrors.Validation("status must be issued or redeemed")
	}
	show, err := s.showtimes.FindByID(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, apperrors.ErrShowtimeNotFound
	}
	rows, err := s.repo.ShowtimeTickets(ctx, showtimeID, status)
	if err != nil {
		return nil, err
	}
	out := make([]dto.StaffTicketResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.StaffTicketResponse{
			ID:        r.ID,
			BookingID: r.BookingID,
			SeatLabel: dto.SeatLabel(r.RowLabel, r.ColNumber),
			SeatType:  r.SeatType,
			Status:    r.Status,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

// BoxOfficeDay settles the counter: how many walk-in sales the register took.
func (s *reportService) BoxOfficeDay(ctx context.Context, date string) (*dto.BoxOfficeDayResponse, error) {
	day := time.Now()
	if date != "" {
		parsed, err := time.ParseInLocation(dto.DateLayout, date, s.location)
		if err != nil {
			return nil, apperrors.Validation("date must follow format YYYY-MM-DD")
		}
		day = parsed
	}
	label, from, to := s.dayBounds(day)
	count, total, err := s.repo.CounterSalesDay(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return &dto.BoxOfficeDayResponse{Date: label, Count: count, Total: total}, nil
}

// AdminOverview is the one-call admin dashboard: today computed live (not
// waiting on closeDay), the last 7 closed days, what's left to show today,
// and operational alerts that would otherwise only surface by manually
// filtering /admin/audit-logs or /admin/batch/jobs.
func (s *reportService) AdminOverview(ctx context.Context) (*dto.AdminOverviewResponse, error) {
	now := time.Now().In(s.location)
	todayLabel, todayFrom, todayTo := s.dayBounds(now)

	live, err := s.repo.LiveDayAggregate(ctx, todayFrom, todayTo)
	if err != nil {
		return nil, err
	}
	today := dto.DailyAggregateResponse{
		ReportDate:    todayLabel,
		TotalRevenue:  live.TotalRevenue,
		TicketsSold:   live.TicketsSold,
		SeatsSold:     live.SeatsSold,
		Capacity:      live.Capacity,
		OccupancyRate: live.OccupancyRate,
		UpdatedAt:     now,
	}

	_, sevenDaysAgo, _ := s.dayBounds(now.AddDate(0, 0, -6))
	pastAggs, err := s.repo.DailyAggregates(ctx, sevenDaysAgo.Format(dto.DateLayout), todayLabel)
	if err != nil {
		return nil, err
	}
	last7 := make([]dto.DailyAggregateResponse, 0, len(pastAggs))
	for i := range pastAggs {
		last7 = append(last7, *newDailyAggregateResponse(&pastAggs[i]))
	}

	board, err := s.repo.ShowtimeBoard(ctx, now, todayTo)
	if err != nil {
		return nil, err
	}
	upcoming := make([]dto.StaffShowtimeResponse, 0, len(board))
	for _, r := range board {
		upcoming = append(upcoming, dto.StaffShowtimeResponse{
			ID: r.ID, MovieTitle: r.MovieTitle, HallName: r.HallName, StartAt: r.StartAt, EndAt: r.EndAt,
			Status: r.Status, Capacity: r.Capacity, Held: r.Held, Sold: r.Sold,
			Available: r.Capacity - r.Held - r.Sold, CheckedIn: r.CheckedIn,
		})
	}

	alerts, err := s.operationalAlerts(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.AdminOverviewResponse{Today: today, Last7Days: last7, UpcomingShowtimes: upcoming, Alerts: *alerts}, nil
}

func (s *reportService) operationalAlerts(ctx context.Context) (*dto.AdminAlertsResponse, error) {
	stuck, err := s.payments.StuckRefunds(ctx, stuckAlertAttempts, alertListLimit)
	if err != nil {
		return nil, err
	}
	stuckOut := make([]dto.StuckRefundAlert, 0, len(stuck))
	for _, p := range stuck {
		stuckOut = append(stuckOut, dto.StuckRefundAlert{
			PaymentID: p.ID, BookingID: p.BookingID, Attempts: p.RefundAttempts,
			Amount: p.Amount, LastError: derefString(p.LastError),
		})
	}

	failedJobs, err := s.batchJobs.RecentFailed(ctx, time.Now().Add(-24*time.Hour), alertListLimit)
	if err != nil {
		return nil, err
	}
	jobsOut := make([]dto.FailedJobAlert, 0, len(failedJobs))
	for _, j := range failedJobs {
		jobsOut = append(jobsOut, dto.FailedJobAlert{
			ID: j.ID, JobName: j.JobName, ErrorMessage: j.ErrorMessage, StartedAt: j.StartedAt,
		})
	}

	givenUp, err := s.bookings.GivenUpEmails(ctx, alertListLimit)
	if err != nil {
		return nil, err
	}
	emailsOut := make([]dto.GivenUpEmailAlert, 0, len(givenUp))
	for _, b := range givenUp {
		emailsOut = append(emailsOut, dto.GivenUpEmailAlert{
			BookingID: b.ID, Attempts: b.EmailAttempts, CreatedAt: b.CreatedAt,
		})
	}

	return &dto.AdminAlertsResponse{StuckRefunds: stuckOut, FailedJobs: jobsOut, GivenUpEmails: emailsOut}, nil
}

// StaffOverview composes the existing staff board and box office numbers
// with one derived count (tickets sold but not yet scanned), so the floor
// app can land on a single call instead of two.
func (s *reportService) StaffOverview(ctx context.Context, date string) (*dto.StaffOverviewResponse, error) {
	board, err := s.StaffBoard(ctx, date)
	if err != nil {
		return nil, err
	}
	boxOffice, err := s.BoxOfficeDay(ctx, date)
	if err != nil {
		return nil, err
	}
	awaiting := 0
	for _, sh := range board.Showtimes {
		awaiting += sh.Sold - sh.CheckedIn
	}
	return &dto.StaffOverviewResponse{
		Date: board.Date, Showtimes: board.Showtimes,
		CounterSalesCount: boxOffice.Count, CounterSalesTotal: boxOffice.Total,
		AwaitingCheckin: awaiting,
	}, nil
}

func newDailyAggregateResponse(a *models.DailyAggregate) *dto.DailyAggregateResponse {
	return &dto.DailyAggregateResponse{
		ReportDate:    a.ReportDate.Format(dto.DateLayout),
		TotalRevenue:  a.TotalRevenue,
		TicketsSold:   a.TicketsSold,
		SeatsSold:     a.SeatsSold,
		Capacity:      a.Capacity,
		OccupancyRate: a.OccupancyRate,
		Breakdown:     a.Breakdown,
		UpdatedAt:     a.UpdatedAt,
	}
}

// AdminStats is the dashboard's four headline counts. Deliberately separate from
// AdminOverview: overview answers "how is today going", stats answers "how big is
// the catalogue" — different cadence, different cost, and the tiles must render
// even when the overview's alert queries are slow. Read-only: no transaction, no
// audit row, no bumpCatalog.
func (s *reportService) AdminStats(ctx context.Context) (*dto.AdminStatsResponse, error) {
	row, err := s.repo.EntityCounts(ctx)
	if err != nil {
		return nil, err
	}
	return &dto.AdminStatsResponse{
		Movies:    row.Movies,
		Showtimes: row.Showtimes,
		Bookings:  row.Bookings,
		Users:     row.Users,
	}, nil
}
