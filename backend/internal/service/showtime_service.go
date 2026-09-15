package service

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"gorm.io/gorm"
)

type ShowtimeService interface {
	Create(ctx context.Context, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error)
	Update(ctx context.Context, id string, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error)
	Delete(ctx context.Context, id string) error
	ListByMovie(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error)
	ListByDate(ctx context.Context, date string) ([]dto.ShowtimeListItem, error)
	SeatMap(ctx context.Context, showtimeID string) (*dto.SeatMapResponse, error)
	OpenShowtime(ctx context.Context, id string) (*dto.ShowtimeResponse, error)
}

func (s *showtimeService) OpenShowtime(ctx context.Context, id string) (*dto.ShowtimeResponse, error) {
	row, err := s.onSale(ctx, id)
	if err != nil {
		return nil, err
	}
	return &dto.ShowtimeResponse{
		ID:         row.ID,
		MovieID:    row.MovieID,
		MovieTitle: row.MovieTitle,
		AgeRating:  row.AgeRating,
		HallID:     row.HallID,
		HallName:   row.HallName,
		StartAt:    row.StartAt,
		EndAt:      row.EndAt,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (s *showtimeService) onSale(ctx context.Context, id string) (*repository.ShowtimeRow, error) {
	row, err := s.showtime.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, apperrors.ErrShowtimeNotFound
	}
	if row.Status != models.ShowtimeOpen || !row.StartAt.After(time.Now()) || row.MovieStatus != models.MovieStatusShowing {
		return nil, apperrors.ErrShowtimeNotOpen
	}
	return row, nil
}

type showtimeService struct {
	db       *gorm.DB
	showtime *repository.ShowtimeRepository
	hall     *repository.HallRepository
	movie    repository.MovieRepository
	cleanup  time.Duration
	location *time.Location
	cache    *cache.Cache
	cacheTTL time.Duration
}

// NewShowtimeService optionally caches public showtime listings; a nil cache disables it.
func NewShowtimeService(db *gorm.DB, showtime *repository.ShowtimeRepository, hall *repository.HallRepository, movie repository.MovieRepository,
	cleanupMinutes int, location *time.Location, c *cache.Cache, cacheTTL time.Duration) ShowtimeService {
	return &showtimeService{
		db:       db,
		showtime: showtime,
		hall:     hall,
		movie:    movie,
		cleanup:  time.Duration(cleanupMinutes) * time.Minute,
		location: location,
		cache:    c,
		cacheTTL: cacheTTL,
	}
}

// endOf share-locks the movie, so ending it or changing its duration waits for
// this transaction and then sees the showtime.
func (s *showtimeService) endOf(ctx context.Context, tx *gorm.DB, movieID string, start time.Time) (time.Time, error) {
	movie, err := s.movie.LockForShare(ctx, tx, movieID)
	if err != nil {
		return time.Time{}, err
	}
	if movie.Status != models.MovieStatusShowing {
		return time.Time{}, apperrors.ErrMovieNotShowing
	}
	if movie.Duration <= 0 {
		return time.Time{}, apperrors.Validation("movie duration must be positive")
	}
	return start.Add(time.Duration(movie.Duration) * time.Minute), nil
}

func (s *showtimeService) Create(ctx context.Context, req dto.ShowtimeRequest) (*dto.ShowtimeResponse, error) {
	hall, err := s.hall.FindByID(ctx, req.HallID)
	if err != nil {
		return nil, err
	}
	if hall == nil {
		return nil, apperrors.ErrHallNotFound
	}
	if !hall.Active {
		return nil, apperrors.ErrHallInactive
	}

	start := req.StartAt.UTC()
	if !start.After(time.Now()) {
		return nil, apperrors.Validation("start_at must be in the future")
	}

	showtime := &models.Showtime{
		MovieID: req.MovieID,
		HallID:  req.HallID,
		StartAt: start,
		Status:  models.ShowtimeOpen,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock order: the hall's scheduling lock, then the movie row (shared).
		if err := s.showtime.LockHall(tx, req.HallID); err != nil {
			return err
		}
		end, err := s.endOf(ctx, tx, req.MovieID, start)
		if err != nil {
			return err
		}
		showtime.EndAt = end
		overlaps, err := s.showtime.OverlapCount(tx, req.HallID, start, end, s.cleanup, "")
		if err != nil {
			return err
		}
		if overlaps > 0 {
			return apperrors.ErrShowtimeOverlap
		}
		if err := s.showtime.Create(tx, showtime); err != nil {
			if apperrors.IsExclusionViolation(err) {
				return apperrors.ErrShowtimeOverlap
			}
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
	bumpCatalog(ctx, s.cache)
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
	hall, err := s.hall.FindByID(ctx, req.HallID)
	if err != nil {
		return nil, err
	}
	if hall == nil {
		return nil, apperrors.ErrHallNotFound
	}

	start := req.StartAt.UTC().Truncate(time.Microsecond) // DB precision, so an unchanged time compares equal
	// Status-only change: closing is always allowed, even once the showtime started
	// or its movie ended; reopening is checked under the locks.
	statusOnly := req.MovieID == row.MovieID && req.HallID == row.HallID && start.Equal(row.StartAt)
	if !statusOnly && !start.After(time.Now()) {
		return nil, apperrors.Validation("start_at must be in the future")
	}

	before := map[string]any{"movie_id": row.MovieID, "hall_id": row.HallID,
		"start_at": row.StartAt, "end_at": row.EndAt, "status": row.Status}

	var (
		end    time.Time
		status string
	)
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Lock order shared with hall layout changes: hall scheduling locks (sorted), the showtime
		// row, then the movie row (shared). Holds share-lock the showtime row, so this waits for them.
		halls := []string{row.HallID}
		if req.HallID != row.HallID {
			halls = append(halls, req.HallID)
			slices.Sort(halls)
		}
		for _, hallID := range halls {
			if err := s.showtime.LockHall(tx, hallID); err != nil {
				return err
			}
		}
		current, err := s.showtime.LockForUpdate(tx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrShowtimeNotFound
		}
		if current.HallID != row.HallID {
			return apperrors.ErrShowtimeChanged // moved by a concurrent update: its hall is not locked
		}
		if statusOnly && (current.MovieID != row.MovieID || !current.StartAt.Equal(row.StartAt)) {
			return apperrors.ErrShowtimeChanged
		}

		status = current.Status
		if req.Status != "" {
			status = req.Status
		}
		hallChanged := false
		if statusOnly {
			end = current.EndAt
			if status == models.ShowtimeOpen && current.Status != models.ShowtimeOpen {
				movie, err := s.movie.LockForShare(ctx, tx, current.MovieID)
				if err != nil {
					return err
				}
				if movie.Status != models.MovieStatusShowing || !current.StartAt.After(time.Now()) || !hall.Active {
					return apperrors.ErrShowtimeReopenLocked
				}
			}
		} else {
			if !hall.Active {
				return apperrors.ErrHallInactive
			}
			if end, err = s.endOf(ctx, tx, req.MovieID, start); err != nil {
				return err
			}
			hallChanged = current.HallID != req.HallID
			if hallChanged {
				// The seat grid is rebuilt for the new hall, only possible while no
				// booking of any status points at the old seats.
				has, err := s.showtime.ShowtimeHasBookings(tx, id)
				if err != nil {
					return err
				}
				if has {
					return apperrors.ErrShowtimeHallLocked
				}
			} else if current.MovieID != req.MovieID || !current.StartAt.Equal(start) {
				// Held and sold tickets name this movie and time: stop sales by
				// closing the showtime instead.
				live, err := s.showtime.ShowtimeHasLiveBookings(tx, id)
				if err != nil {
					return err
				}
				if live {
					return apperrors.ErrShowtimeScheduleLocked
				}
			}

			overlaps, err := s.showtime.OverlapCount(tx, req.HallID, start, end, s.cleanup, id)
			if err != nil {
				return err
			}
			if overlaps > 0 {
				return apperrors.ErrShowtimeOverlap
			}
		}

		showtime := &models.Showtime{ID: id}
		showtime.MovieID = req.MovieID
		showtime.HallID = req.HallID
		showtime.StartAt = start
		showtime.EndAt = end
		showtime.Status = status
		if err := s.showtime.Update(tx, showtime); err != nil {
			if apperrors.IsExclusionViolation(err) {
				return apperrors.ErrShowtimeOverlap
			}
			return err
		}
		if hallChanged {
			if err := s.showtime.DeleteSeatStates(tx, id); err != nil {
				return err
			}
			seats, err := s.hall.SeatsByHall(ctx, req.HallID)
			if err != nil {
				return err
			}
			if err := s.showtime.CreateSeatStates(tx, id, seats); err != nil {
				return err
			}
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
	bumpCatalog(ctx, s.cache)

	return &dto.ShowtimeResponse{
		ID:      id,
		MovieID: req.MovieID,
		HallID:  req.HallID,
		StartAt: start,
		EndAt:   end,
		Status:  status,
	}, nil
}

func (s *showtimeService) Delete(ctx context.Context, id string) error {
	row, err := s.showtime.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if row == nil {
		return apperrors.ErrShowtimeNotFound
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock first, check after: an in-flight hold share-locks the row, so its booking
		// is committed and seen by the check; a later hold finds the showtime gone.
		current, err := s.showtime.LockForUpdate(tx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrShowtimeNotFound
		}
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
	}); err != nil {
		return err
	}
	bumpCatalog(ctx, s.cache)
	return nil
}

func (s *showtimeService) ListByMovie(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error) {
	movie, err := s.movie.FindByID(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if movie.Status != models.MovieStatusShowing {
		return []dto.ShowtimeListItem{}, nil
	}

	return s.pickDay(ctx, movieID, date)
}

func (s *showtimeService) ListByDate(ctx context.Context, date string) ([]dto.ShowtimeListItem, error) {
	return s.pickDay(ctx, "", date)
}

func (s *showtimeService) pickDay(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error) {
	key := showtimeListKey(catalogGeneration(ctx, s.cache), movieID, date)
	if raw, ok, err := s.cache.Get(ctx, key); err == nil && ok {
		var cached []dto.ShowtimeListItem
		if json.Unmarshal([]byte(raw), &cached) == nil {
			return cached, nil
		}
	}
	result, err := s.pickDayUncached(ctx, movieID, date)
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, key, string(raw), s.cacheTTL)
	}
	return result, nil
}

func (s *showtimeService) pickDayUncached(ctx context.Context, movieID, date string) ([]dto.ShowtimeListItem, error) {
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
			ID:         row.ID,
			MovieID:    row.MovieID,
			MovieTitle: row.MovieTitle,
			AgeRating:  row.AgeRating,
			HallID:     row.HallID,
			HallName:   row.HallName,
			StartAt:    row.StartAt,
			EndAt:      row.EndAt,
			Status:     row.Status,
			FromPrice:  row.FromPrice,
		})
	}
	return result, nil
}

func (s *showtimeService) SeatMap(ctx context.Context, showtimeID string) (*dto.SeatMapResponse, error) {
	row, err := s.onSale(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	seats, err := s.showtime.SeatMap(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	hall, err := s.hall.FindByID(ctx, row.HallID)
	if err != nil {
		return nil, err
	}

	response := &dto.SeatMapResponse{
		ShowtimeID:     showtimeID,
		MovieID:        row.MovieID,
		MovieTitle:     row.MovieTitle,
		AgeRating:      row.AgeRating,
		HallID:         row.HallID,
		HallName:       row.HallName,
		StartAt:        row.StartAt,
		EndAt:          row.EndAt,
		Status:         row.Status,
		ScreenPosition: hall.ScreenPosition,
		AisleAfterCols: hall.AisleAfterCols,
		Prices:         map[string]int64{},
		Seats:          make([]dto.SeatMapSeat, 0, len(seats)),
	}
	for _, seat := range seats {
		status := seat.SeatStatus
		if status == "" {
			status = models.SeatStatusAvailable
		}
		response.Prices[seat.SeatType] = seat.Price
		response.Seats = append(response.Seats, dto.SeatMapSeat{
			ID:             seat.SeatID,
			ShowtimeSeatID: seat.SeatShowtimeID,
			Label:          dto.SeatLabel(seat.RowLabel, seat.ColNumber),
			RowLabel:       seat.RowLabel,
			Col:            seat.ColNumber,
			SeatType:       seat.SeatType,
			IsGap:          seat.IsGap,
			ColSpan:        seat.ColSpan,
			Status:         status,
			Price:          seat.Price,
		})
	}
	return response, nil
}
