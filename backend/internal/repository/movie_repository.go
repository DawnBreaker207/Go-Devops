package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// MovieRepository accesses the movies table. Writes take the database (or
// transaction) to execute on so the service can persist audit rows in the
// same transaction.
type MovieRepository interface {
	Create(ctx context.Context, db *gorm.DB, movie *models.Movie) error
	Update(ctx context.Context, db *gorm.DB, movie *models.Movie) error
	Delete(ctx context.Context, db *gorm.DB, id string) error
	FindByID(ctx context.Context, id string) (*models.Movie, error)
	List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]models.Movie, int64, error)

	// LockForUpdate locks a movie row to change it; LockForShare to schedule a
	// showtime of it. Both answer ErrMovieNotFound for a missing movie.
	LockForUpdate(ctx context.Context, tx *gorm.DB, id string) (*models.Movie, error)
	LockForShare(ctx context.Context, tx *gorm.DB, id string) (*models.Movie, error)
	// HasUpcomingShowtimes reports whether showtimes of the movie are still to
	// start: open ones, and closed ones too with includeClosed.
	HasUpcomingShowtimes(ctx context.Context, tx *gorm.DB, id string, includeClosed bool) (bool, error)
}

type movieRepository struct {
	db *gorm.DB
}

func NewMovieRepository(db *gorm.DB) MovieRepository {
	return &movieRepository{db: db}
}

func (r *movieRepository) Create(ctx context.Context, db *gorm.DB, movie *models.Movie) error {
	if err := db.WithContext(ctx).Create(movie).Error; err != nil {
		return fmt.Errorf("create movie: %w", err)
	}
	return nil
}

func (r *movieRepository) Update(ctx context.Context, db *gorm.DB, movie *models.Movie) error {
	if err := db.WithContext(ctx).Save(movie).Error; err != nil {
		return fmt.Errorf("update movie: %w", err)
	}
	return nil
}

func (r *movieRepository) Delete(ctx context.Context, db *gorm.DB, id string) error {
	result := db.WithContext(ctx).Delete(&models.Movie{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete movie: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrMovieNotFound
	}
	return nil
}

func (r *movieRepository) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	var movie models.Movie
	if err := r.db.WithContext(ctx).First(&movie, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrMovieNotFound
		}
		return nil, fmt.Errorf("find movie by id: %w", err)
	}
	return &movie, nil
}

func (r *movieRepository) LockForUpdate(ctx context.Context, tx *gorm.DB, id string) (*models.Movie, error) {
	return r.lock(ctx, tx, id, "UPDATE")
}

func (r *movieRepository) LockForShare(ctx context.Context, tx *gorm.DB, id string) (*models.Movie, error) {
	return r.lock(ctx, tx, id, "SHARE")
}

func (r *movieRepository) lock(ctx context.Context, tx *gorm.DB, id, strength string) (*models.Movie, error) {
	var movie models.Movie
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: strength}).First(&movie, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrMovieNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock movie: %w", err)
	}
	return &movie, nil
}

func (r *movieRepository) HasUpcomingShowtimes(ctx context.Context, tx *gorm.DB, id string, includeClosed bool) (bool, error) {
	query := `SELECT 1 FROM showtimes WHERE movie_id = ? AND deleted_at IS NULL AND start_at > NOW()`
	args := []any{id}
	if !includeClosed {
		query += ` AND status = ?`
		args = append(args, models.ShowtimeOpen)
	}
	var found []int
	if err := tx.WithContext(ctx).Raw(query+` LIMIT 1`, args...).Scan(&found).Error; err != nil {
		return false, fmt.Errorf("find upcoming showtimes of movie: %w", err)
	}
	return len(found) > 0, nil
}

func (r *movieRepository) List(ctx context.Context, query dto.MovieListQuery, includeDrafts bool) ([]models.Movie, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Movie{})

	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}
	if !includeDrafts {
		tx = tx.Where("status <> ?", models.MovieStatusDraft)
	}

	if search := strings.TrimSpace(query.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(director) LIKE ? OR LOWER(genre) LIKE ?", pattern, pattern, pattern)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count movies: %w", err)
	}

	movies := make([]models.Movie, 0, query.PageSize)
	if err := tx.
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset(query.Offset()).
		Find(&movies).Error; err != nil {
		return nil, 0, fmt.Errorf("list movies: %w", err)
	}

	return movies, total, nil
}
