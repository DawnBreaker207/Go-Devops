package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

const DateLayout = "2006-01-02"

type MovieRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=255" example:"Inception"`
	Genre       string `json:"genre" binding:"required,max=100" example:"Sci-Fi"`
	Duration    int    `json:"duration" binding:"required,min=1,max=600" example:"148"`
	Director    string `json:"director" binding:"required,max=255" example:"Christopher Nolan"`
	Description string `json:"description" binding:"omitempty,max=5000"`
	PosterURL   string `json:"poster_url" binding:"omitempty,url,max=512"`
	TrailerURL  string `json:"trailer_url" binding:"omitempty,url,max=512" example:"https://www.youtube.com/watch?v=YoHD9XEInc0"`
	Cast        string `json:"cast" binding:"omitempty,max=2000" example:"Leonardo DiCaprio, Joseph Gordon-Levitt"`
	// AgeRating is Vietnamese film classification; empty defaults to P.
	AgeRating   string `json:"age_rating" binding:"omitempty,oneof=P K T13 T16 T18" example:"T18"`
	ReleaseDate string `json:"release_date" binding:"required,datetime=2006-01-02" example:"2010-07-16"`
	Status      string `json:"status" binding:"required,oneof=draft coming_soon showing ended" example:"showing"`
}

// Binding validates the format, so errors only occur on direct calls.
func (r MovieRequest) ParseReleaseDate() (time.Time, error) {
	return time.Parse(DateLayout, r.ReleaseDate)
}

// Customers never see drafts.
type MovieListQuery struct {
	PageQuery
	Status string `form:"status" binding:"omitempty,oneof=draft coming_soon showing ended"`
	Genre  string `form:"genre" binding:"omitempty,max=100"`
	Sort   string `form:"sort" binding:"omitempty,oneof=release_date title created_at"`
	Order  string `form:"order" binding:"omitempty,oneof=asc desc" example:"desc"`
}

type MovieResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Genre       string    `json:"genre"`
	Duration    int       `json:"duration"`
	Director    string    `json:"director"`
	Description string    `json:"description"`
	PosterURL   string    `json:"poster_url"`
	TrailerURL  string    `json:"trailer_url"`
	Cast        string    `json:"cast"`
	AgeRating   string    `json:"age_rating"`
	ReleaseDate string    `json:"release_date" example:"2010-07-16"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewMovieResponse(movie *models.Movie) MovieResponse {
	return MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Genre:       movie.Genre,
		Duration:    movie.Duration,
		Director:    movie.Director,
		Description: movie.Description,
		PosterURL:   movie.PosterURL,
		TrailerURL:  movie.TrailerURL,
		Cast:        movie.Cast,
		AgeRating:   movie.AgeRating,
		ReleaseDate: movie.ReleaseDate.Format(DateLayout),
		Status:      movie.Status,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}
}

func NewMovieResponses(movies []models.Movie) []MovieResponse {
	result := make([]MovieResponse, 0, len(movies))
	for i := range movies {
		result = append(result, NewMovieResponse(&movies[i]))
	}
	return result
}
