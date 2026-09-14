package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// canSeeDrafts: draft movies are for admin/staff only.
func canSeeDrafts(c *gin.Context) bool {
	role := middleware.CurrentUserRole(c)
	return role == models.RoleAdmin || role == models.RoleStaff
}

// MovieHandler handles movie requests.
type MovieHandler struct {
	movieService service.MovieService
}

func NewMovieHandler(movieService service.MovieService) *MovieHandler {
	return &MovieHandler{movieService: movieService}
}

// List godoc
//
//	@Summary		List movies (paginated)
//	@Tags			movies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			search		query		string	false	"Search by title, director, or genre"
//	@Param			status		query		string	false	"draft | showing | ended (customers never see drafts)"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.MovieResponse}}
//	@Failure		400			{object}	response.Body
//	@Failure		401			{object}	response.Body
//	@Router			/movies [get]
func (h *MovieHandler) List(c *gin.Context) {
	var query dto.MovieListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()

	movies, total, err := h.movieService.List(c.Request.Context(), query, canSeeDrafts(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.List(c, movies, query.Page, query.PageSize, total)
}

// Detail godoc
//
//	@Summary		Get movie details
//	@Tags			movies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Movie ID"
//	@Success		200	{object}	response.Body{data=dto.MovieResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/movies/{id} [get]
func (h *MovieHandler) Detail(c *gin.Context) {
	movie, err := h.movieService.GetByID(c.Request.Context(), c.Param("id"), canSeeDrafts(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, movie)
}

// Create godoc
//
//	@Summary		Create a new movie
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.MovieRequest	true	"Movie details"
//	@Success		201		{object}	response.Body{data=dto.MovieResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Router			/movies [post]
func (h *MovieHandler) Create(c *gin.Context) {
	var req dto.MovieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	movie, err := h.movieService.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, movie)
}

// Update godoc
//
//	@Summary		Update a movie
//	@Tags			movies
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Movie ID"
//	@Param			payload	body		dto.MovieRequest	true	"Movie details"
//	@Success		200		{object}	response.Body{data=dto.MovieResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/movies/{id} [put]
func (h *MovieHandler) Update(c *gin.Context) {
	var req dto.MovieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	movie, err := h.movieService.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, movie)
}

// Delete godoc
//
//	@Summary		Delete a movie
//	@Tags			movies
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Movie ID"
//	@Success		200	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/movies/{id} [delete]
func (h *MovieHandler) Delete(c *gin.Context) {
	if err := h.movieService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}

	response.NoContentOK(c, "deleted")
}
