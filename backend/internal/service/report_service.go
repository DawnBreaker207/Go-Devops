package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// ReportService closes business days (F21 closeDay) and feeds the staff
// board (the MVP part of F17). Days are local days of the configured timezone.
type ReportService interface {
	CloseDay(ctx context.Context, day time.Time) (*dto.DailyAggregateResponse, error)
	DailyReport(ctx context.Context, from, to string) (*dto.DailyReportResponse, error)
	StaffBoard(ctx context.Context, date string) (*dto.StaffBoardResponse, error)
	ShowtimeTickets(ctx context.Context, showtimeID, status string) ([]dto.StaffTicketResponse, error)
}

type reportService struct {
	repo      repository.ReportRepository
	showtimes *repository.ShowtimeRepository
	location  *time.Location
}

func NewReportService(repo repository.ReportRepository, showtimes *repository.ShowtimeRepository, location *time.Location) ReportService {
	return &reportService{repo: repo, showtimes: showtimes, location: location}
}

// dayBounds returns the local date and its [from, to) instants.
func (s *reportService) dayBounds(day time.Time) (string, time.Time, time.Time) {
	d := day.In(s.location)
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, s.location)
	return from.Format(dto.DateLayout), from, from.AddDate(0, 0, 1)
}

// CloseDay writes the daily_aggregates row of the local day containing day.
// Running it again replaces the numbers of that row (E-B2, T24).
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

// DailyReport returns the closed days in [from, to] (default: the last 7 days)
// for admins (F17 minimal, T15). Numbers are what closeDay last wrote to
// daily_aggregates: CONFIRMED money by payment time only (I4); run closeDay to
// refresh today.
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
		return nil, apperrors.Validation("from must not be after to") // E-D1
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

// StaffBoard lists the showtimes of a day (default today) with seat counts.
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

// ShowtimeTickets lists the sold tickets of a showtime; status "issued" is the
// list still waiting at the gate.
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
