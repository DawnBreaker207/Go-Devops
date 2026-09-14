package service

import (
	"context"
	"strings"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"gorm.io/gorm"
)

type MovieService interface {
	// includeDrafts is true for admin/staff; customers never see draft movies.
	List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]dto.MovieResponse, int64, error)
	GetByID(ctx context.Context, id string, includeDrafts bool) (*dto.MovieResponse, error)
	Create(ctx context.Context, req dto.MovieRequest) (*dto.MovieResponse, error)
	Update(ctx context.Context, id string, req dto.MovieRequest) (*dto.MovieResponse, error)
	Delete(ctx context.Context, id string) error
}

type movieService struct {
	db        *gorm.DB
	movieRepo repository.MovieRepository
}

func NewMovieService(db *gorm.DB, movieRepo repository.MovieRepository) MovieService {
	return &movieService{db: db, movieRepo: movieRepo}
}

func (s *movieService) List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]dto.MovieResponse, int64, error) {
	movies, total, err := s.movieRepo.List(ctx, query, includeDrafts)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewMovieResponses(movies), total, nil
}

func (s *movieService) GetByID(ctx context.Context, id string, includeDrafts bool) (*dto.MovieResponse, error) {
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !includeDrafts && movie.Status == models.MovieStatusDraft {
		return nil, apperrors.ErrMovieNotFound
	}
	result := dto.NewMovieResponse(movie)
	return &result, nil
}

func (s *movieService) Create(ctx context.Context, req dto.MovieRequest) (*dto.MovieResponse, error) {
	releaseDate, err := req.ParseReleaseDate()
	if err != nil {
		return nil, apperrors.Validation("release_date must follow format YYYY-MM-DD").Wrap(err)
	}

	movie := &models.Movie{
		Title:       strings.TrimSpace(req.Title),
		Genre:       strings.TrimSpace(req.Genre),
		Duration:    req.Duration,
		Director:    strings.TrimSpace(req.Director),
		Description: req.Description,
		PosterURL:   req.PosterURL,
		ReleaseDate: releaseDate,
		Status:      req.Status,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.movieRepo.Create(ctx, tx, movie); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = movie.ID
			rec.After = map[string]any{"title": movie.Title, "status": movie.Status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result := dto.NewMovieResponse(movie)
	return &result, nil
}

// Update refuses to leave "showing" or change the duration while showtimes are still to come:
// sold tickets would point at an unscheduled movie and showtimes would end at the wrong time.
func (s *movieService) Update(ctx context.Context, id string, req dto.MovieRequest) (*dto.MovieResponse, error) {
	releaseDate, err := req.ParseReleaseDate()
	if err != nil {
		return nil, apperrors.Validation("release_date must follow format YYYY-MM-DD").Wrap(err)
	}

	var movie *models.Movie
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the movie, then look at its showtimes: scheduling a showtime
		// share-locks the movie, so none slips in between.
		current, err := s.movieRepo.LockForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Status == models.MovieStatusShowing && req.Status != models.MovieStatusShowing {
			upcoming, err := s.movieRepo.HasUpcomingShowtimes(ctx, tx, id, false)
			if err != nil {
				return err
			}
			if upcoming {
				return apperrors.ErrMovieHasShowtimes
			}
		}
		if current.Duration != req.Duration {
			// Closed showtimes count too: reopened, they would end at the wrong time.
			upcoming, err := s.movieRepo.HasUpcomingShowtimes(ctx, tx, id, true)
			if err != nil {
				return err
			}
			if upcoming {
				return apperrors.ErrMovieDurationLocked
			}
		}

		current.Title = strings.TrimSpace(req.Title)
		current.Genre = strings.TrimSpace(req.Genre)
		current.Duration = req.Duration
		current.Director = strings.TrimSpace(req.Director)
		current.Description = req.Description
		current.PosterURL = req.PosterURL
		current.ReleaseDate = releaseDate
		current.Status = req.Status
		if err := s.movieRepo.Update(ctx, tx, current); err != nil {
			return err
		}
		movie = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.After = map[string]any{"title": current.Title, "status": current.Status}
			return audit.In(ctx, tx, rec)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result := dto.NewMovieResponse(movie)
	return &result, nil
}

// Delete soft-deletes a movie; it is refused while open showtimes are still to come.
func (s *movieService) Delete(ctx context.Context, id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := s.movieRepo.LockForUpdate(ctx, tx, id); err != nil {
			return err
		}
		upcoming, err := s.movieRepo.HasUpcomingShowtimes(ctx, tx, id, false)
		if err != nil {
			return err
		}
		if upcoming {
			return apperrors.ErrMovieHasShowtimes
		}
		if err := s.movieRepo.Delete(ctx, tx, id); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.After = map[string]any{"deleted": true}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}
