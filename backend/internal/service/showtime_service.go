package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"gorm.io/gorm"
)

// ShowtimeService schedules and serves showtimes.
type ShowtimeService interface {
	Create(ctx context.Context, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error)
	Update(ctx context.Context, id string, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error)
	Delete(ctx context.Context, id string) error
	ListByMovie(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error)
	SeatMap(ctx context.Context, showtimeID string) (*dto.SeatMapResponse, error)
}

type showtimeService struct {
	db       *gorm.DB
	showtime *repository.ShowtimeRepository
	hall     *repository.HallRepository
	movie    repository.MovieRepository
	cleanup  time.Duration
	location *time.Location
}

func NewShowtimeService(db *gorm.DB, showtime *repository.ShowtimeRepository, hall *repository.HallRepository, movie repository.MovieRepository, cleanupMinutes int, location *time.Location) ShowtimeService {
	return &showtimeService{
		db:       db,
		showtime: showtime,
		hall:     hall,
		movie:    movie,
		cleanup:  time.Duration(cleanupMinutes) * time.Minute,
		location: location,
	}
}

func (s *showtimeService) Create(ctx context.Context, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error) {
	movie, err := s.movie.FindByID(ctx, req.MovieID)
	if err != nil {
		return nil, err
	}
	if movie.Status != models.MovieStatusShowing {
		return nil, apperrors.ErrMovieNotShowing
	}
	if movie.Duration <= 0 {
		return nil, apperrors.Validation("movie duration must be positive")
	}
	if _, err := s.hall.FindByID(ctx, req.HallID); err != nil {
		return nil, err
	}

	start := req.StartAt.UTC()
	if !start.After(time.Now()) {
		return nil, apperrors.Validation("start_at must be in the future")
	}
	end := start.Add(time.Duration(movie.Duration) * time.Minute)
	effectiveEnd := end.Add(s.cleanup)

	showtime := &models.Showtime{
		MovieID: req.MovieID,
		HallID:  req.HallID,
		StartAt: start,
		EndAt:   end,
		Status:  models.ShowtimeOpen,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.showtime.LockHall(tx, req.HallID); err != nil {
			return err
		}
		overlaps, err := s.showtime.OverlapCount(tx, req.HallID, start, effectiveEnd, "")
		if err != nil {
			return err
		}
		if overlaps > 0 {
			return apperrors.ErrShowtimeOverlap
		}
		if err := s.showtime.Create(tx, showtime); err != nil {
			return err
		}
		seats, err := s.hall.SeatsByHall(ctx, req.HallID)
		if err != nil {
			return err
		}
		if err := s.showtime.CreateSeatStates(tx, showtime.ID, seats); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = showtime.ID
			rec.After = map[string]any{"movie_id": showtime.MovieID, "hall_id": showtime.HallID,
				"start_at": showtime.StartAt, "end_at": showtime.EndAt}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := &dto.ShowtimeResponse{
		ID:      showtime.ID,
		MovieID: showtime.MovieID,
		HallID:  showtime.HallID,
		StartAt: showtime.StartAt,
		EndAt:   showtime.EndAt,
		Status:  showtime.Status,
	}
	return result, nil
}

func (s *showtimeService) Update(ctx context.Context, id string, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error) {
	row, err := s.showtime.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apperrors.ErrShowtimeNotFound
	}
	movie, err := s.movie.FindByID(ctx, req.MovieID)
	if err != nil {
		return nil, err
	}
	if movie.Status != models.MovieStatusShowing {
		return nil, apperrors.ErrMovieNotShowing
	}
	if movie.Duration <= 0 {
		return nil, apperrors.Validation("movie duration must be positive")
	}
	if _, err := s.hall.FindByID(ctx, req.HallID); err != nil {
		return nil, err
	}

	start := req.StartAt.UTC()
	if !start.After(time.Now()) {
		return nil, apperrors.Validation("start_at must be in the future")
	}
	end := start.Add(time.Duration(movie.Duration) * time.Minute)
	effectiveEnd := end.Add(s.cleanup)

	before := map[string]any{"movie_id": row.MovieID, "hall_id": row.HallID,
		"start_at": row.StartAt, "end_at": row.EndAt, "status": row.Status}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.showtime.LockHall(tx, req.HallID); err != nil {
			return err
		}
		overlaps, err := s.showtime.OverlapCount(tx, req.HallID, start, effectiveEnd, id)
		if err != nil {
			return err
		}
		if overlaps > 0 {
			return apperrors.ErrShowtimeOverlap
		}
		showtime := &models.Showtime{ID: id}
		showtime.MovieID = req.MovieID
		showtime.HallID = req.HallID
		showtime.StartAt = start
		showtime.EndAt = end
		if req.Status != "" {
			showtime.Status = req.Status
		} else {
			showtime.Status = row.Status
		}
		if err := s.showtime.Update(tx, showtime); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = before
			rec.After = map[string]any{"movie_id": showtime.MovieID, "hall_id": showtime.HallID,
				"start_at": showtime.StartAt, "end_at": showtime.EndAt, "status": showtime.Status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := &dto.ShowtimeResponse{
		ID:      id,
		MovieID: req.MovieID,
		HallID:  req.HallID,
		StartAt: start,
		EndAt:   end,
		Status:  req.Status,
	}
	if result.Status == "" {
		result.Status = row.Status
	}
	return result, nil
}

func (s *showtimeService) Delete(ctx context.Context, id string) error {
	row, err := s.showtime.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if row == nil {
		return apperrors.ErrShowtimeNotFound
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		hasBookings, err := s.showtime.ShowtimeHasBookings(tx, id)
		if err != nil {
			return err
		}
		if hasBookings {
			return apperrors.ErrShowtimeHasBookings
		}
		if err := s.showtime.Delete(tx, id); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.Before = map[string]any{"start_at": row.StartAt, "end_at": row.EndAt}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}

func (s *showtimeService) ListByMovie(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error) {
	movie, err := s.movie.FindByID(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if movie.Status != models.MovieStatusShowing {
		return nil, apperrors.ErrMovieNotShowing
	}

	day := time.Now().In(s.location)
	if date != "" {
		parsed, err := time.ParseInLocation(dto.DateLayout, date, s.location)
		if err != nil {
			return nil, apperrors.Validation("date must follow format YYYY-MM-DD")
		}
		day = parsed
	}
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, s.location)
	end := start.AddDate(0, 0, 1)

	rows, err := s.showtime.PickingList(ctx, movieID, start, end, time.Now().In(s.location))
	if err != nil {
		return nil, err
	}
	result := make([]dto.ShowtimeListItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, dto.ShowtimeListItem{
			ID:        row.ID,
			MovieID:   row.MovieID,
			HallID:    row.HallID,
			HallName:  row.HallName,
			StartAt:   row.StartAt,
			EndAt:     row.EndAt,
			Status:    row.Status,
			FromPrice: row.FromPrice,
		})
	}
	return result, nil
}

func (s *showtimeService) SeatMap(ctx context.Context, showtimeID string) (*dto.SeatMapResponse, error) {
	row, err := s.showtime.FindByID(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apperrors.ErrShowtimeNotFound
	}
	if row.Status != models.ShowtimeOpen {
		return nil, apperrors.ErrShowtimeNotOpen
	}
	seats, err := s.showtime.SeatMap(ctx, showtimeID)
	if err != nil {
		return nil, err
	}

	response := &dto.SeatMapResponse{
		ShowtimeID: showtimeID,
		MovieID:    row.MovieID,
		MovieTitle: row.MovieTitle,
		HallID:     row.HallID,
		HallName:   row.HallName,
		StartAt:    row.StartAt,
		EndAt:      row.EndAt,
		Status:     row.Status,
		Prices:     map[string]int64{},
		Seats:      make([]dto.SeatMapSeat, 0, len(seats)),
	}
	for _, seat := range seats {
		status := seat.SeatStatus
		if status == "" {
			status = models.SeatStatusAvailable
		}
		response.Prices[seat.SeatType] = seat.Price
		response.Seats = append(response.Seats, dto.SeatMapSeat{
			ID:       seat.SeatID,
			Label:    dto.SeatLabel(seat.RowLabel, seat.ColNumber),
			RowLabel: seat.RowLabel,
			Col:      seat.ColNumber,
			SeatType: seat.SeatType,
			IsGap:    seat.IsGap,
			Status:   status,
			Price:    seat.Price,
		})
	}
	return response, nil
}