package service

import (
	"context"
	"strings"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// MovieService xu ly nghiep vu quan ly phim.
type MovieService interface {
	List(ctx context.Context, query dto.PageQuery) ([]dto.MovieResponse, int64, error)
	GetByID(ctx context.Context, id string) (*dto.MovieResponse, error)
	Create(ctx context.Context, req dto.MovieRequest) (*dto.MovieResponse, error)
	Update(ctx context.Context, id string, req dto.MovieRequest) (*dto.MovieResponse, error)
	Delete(ctx context.Context, id string) error
}

type movieService struct {
	movieRepo repository.MovieRepository
}

// NewMovieService tao MovieService.
func NewMovieService(movieRepo repository.MovieRepository) MovieService {
	return &movieService{movieRepo: movieRepo}
}

func (s *movieService) List(ctx context.Context, query dto.PageQuery) ([]dto.MovieResponse, int64, error) {
	movies, total, err := s.movieRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewMovieResponses(movies), total, nil
}

func (s *movieService) GetByID(ctx context.Context, id string) (*dto.MovieResponse, error) {
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
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
	if err := s.movieRepo.Create(ctx, movie); err != nil {
		return nil, err
	}

	result := dto.NewMovieResponse(movie)
	return &result, nil
}

func (s *movieService) Update(ctx context.Context, id string, req dto.MovieRequest) (*dto.MovieResponse, error) {
	movie, err := s.movieRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	releaseDate, err := req.ParseReleaseDate()
	if err != nil {
		return nil, apperrors.Validation("release_date must follow format YYYY-MM-DD").Wrap(err)
	}

	movie.Title = strings.TrimSpace(req.Title)
	movie.Genre = strings.TrimSpace(req.Genre)
	movie.Duration = req.Duration
	movie.Director = strings.TrimSpace(req.Director)
	movie.Description = req.Description
	movie.PosterURL = req.PosterURL
	movie.ReleaseDate = releaseDate
	movie.Status = req.Status

	if err := s.movieRepo.Update(ctx, movie); err != nil {
		return nil, err
	}

	result := dto.NewMovieResponse(movie)
	return &result, nil
}

func (s *movieService) Delete(ctx context.Context, id string) error {
	return s.movieRepo.Delete(ctx, id)
}
