package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

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
	List(ctx context.Context, query dto.PageQuery) ([]models.Movie, int64, error)
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

func (r *movieRepository) List(ctx context.Context, query dto.PageQuery) ([]models.Movie, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.Movie{})

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
