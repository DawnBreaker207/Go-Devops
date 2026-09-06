package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// DateLayout la dinh dang ngay dung tren API.
const DateLayout = "2006-01-02"

// MovieRequest la body dung chung cho tao moi va cap nhat phim.
type MovieRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=255" example:"Inception"`
	Genre       string `json:"genre" binding:"required,max=100" example:"Sci-Fi"`
	Duration    int    `json:"duration" binding:"required,min=1,max=600" example:"148"`
	Director    string `json:"director" binding:"required,max=255" example:"Christopher Nolan"`
	Description string `json:"description" binding:"omitempty,max=5000"`
	PosterURL   string `json:"poster_url" binding:"omitempty,url,max=512"`
	ReleaseDate string `json:"release_date" binding:"required,datetime=2006-01-02" example:"2010-07-16"`
	Status      string `json:"status" binding:"required,oneof=draft showing ended" example:"showing"`
}

// ParseReleaseDate doi chuoi ngay sang time.Time.
// Binding da chan dinh dang nen loi o day chi xay ra khi goi truc tiep.
func (r MovieRequest) ParseReleaseDate() (time.Time, error) {
	return time.Parse(DateLayout, r.ReleaseDate)
}

// MovieResponse la phim tra ve cho client.
type MovieResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Genre       string    `json:"genre"`
	Duration    int       `json:"duration"`
	Director    string    `json:"director"`
	Description string    `json:"description"`
	PosterURL   string    `json:"poster_url"`
	ReleaseDate string    `json:"release_date" example:"2010-07-16"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewMovieResponse map model sang DTO.
func NewMovieResponse(movie *models.Movie) MovieResponse {
	return MovieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Genre:       movie.Genre,
		Duration:    movie.Duration,
		Director:    movie.Director,
		Description: movie.Description,
		PosterURL:   movie.PosterURL,
		ReleaseDate: movie.ReleaseDate.Format(DateLayout),
		Status:      movie.Status,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}
}

// NewMovieResponses map danh sach model sang DTO.
func NewMovieResponses(movies []models.Movie) []MovieResponse {
	result := make([]MovieResponse, 0, len(movies))
	for i := range movies {
		result = append(result, NewMovieResponse(&movies[i]))
	}
	return result
}
