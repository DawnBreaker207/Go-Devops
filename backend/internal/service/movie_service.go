package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
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
	cache     *cache.Cache
	cacheTTL  time.Duration
}

// NewMovieService optionally caches public movie reads; a nil cache disables it.
func NewMovieService(db *gorm.DB, movieRepo repository.MovieRepository, c *cache.Cache, ttl time.Duration) MovieService {
	return &movieService{db: db, movieRepo: movieRepo, cache: c, cacheTTL: ttl}
}

func movieKey(id string) string { return "movie:" + id }

// defaultAgeRating: an omitted rating means everybody (P, per GORDP 2022/17).
func defaultAgeRating(rating string) string {
	if rating == "" {
		return "P"
	}
	return rating
}

// movieListCache is what a cached List result carries.
type movieListCache struct {
	Items []dto.MovieResponse `json:"items"`
	Total int64                `json:"total"`
}

func (s *movieService) List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]dto.MovieResponse, int64, error) {
	var key string
	if !includeDrafts {
		key = movieListKey(catalogGeneration(ctx, s.cache), includeDrafts, query.Status, query.Genre,
			query.Sort, query.Order, query.Search, query.Page, query.PageSize)
		if raw, ok, err := s.cache.Get(ctx, key); err == nil && ok {
			var cached movieListCache
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return cached.Items, cached.Total, nil
			}
		}
	}
	movies, total, err := s.movieRepo.List(ctx, query, includeDrafts)
	if err != nil {
		return nil, 0, err
	}
	items := dto.NewMovieResponses(movies)
	if !includeDrafts {
		if raw, err := json.Marshal(movieListCache{Items: items, Total: total}); err == nil {
			_ = s.cache.Set(ctx, key, string(raw), s.cacheTTL)
		}
	}
	return items, total, nil
}

func (s *movieService) GetByID(ctx context.Context, id string, includeDrafts bool) (*dto.MovieResponse, error) {
	if !includeDrafts {
		if raw, ok, err := s.cache.Get(ctx, movieKey(id)); err == nil && ok {
			var cached dto.MovieResponse
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return &cached, nil
			}
		}
	}
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !includeDrafts && movie.Status == models.MovieStatusDraft {
		return nil, apperrors.ErrMovieNotFound
	}
	result := dto.NewMovieResponse(movie)
	if !includeDrafts {
		if raw, err := json.Marshal(result); err == nil {
			s.cache.Set(ctx, movieKey(id), string(raw), s.cacheTTL)
		}
	}
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
		TrailerURL:  req.TrailerURL,
		Cast:        req.Cast,
		AgeRating:   defaultAgeRating(req.AgeRating),
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
	bumpCatalog(ctx, s.cache)

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
		current.TrailerURL = req.TrailerURL
		current.Cast = req.Cast
		current.AgeRating = defaultAgeRating(req.AgeRating)
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
	s.cache.Del(ctx, movieKey(id))
	bumpCatalog(ctx, s.cache)
	return &result, nil
}

// Delete soft-deletes a movie; it is refused while open showtimes are still to come.
func (s *movieService) Delete(ctx context.Context, id string) error {
	if err := s.db.Transaction(func(tx *gorm.DB) error {
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
	}); err != nil {
		return err
	}
	// Only after commit: busting first would let a concurrent reader
	// repopulate the cache with the pre-delete row before it lands.
	s.cache.Del(ctx, movieKey(id))
	bumpCatalog(ctx, s.cache)
	return nil
}
