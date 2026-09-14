package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ShowtimeHandler struct {
	showtimeService service.ShowtimeService
}

func NewShowtimeHandler(showtimeService service.ShowtimeService) *ShowtimeHandler {
	return &ShowtimeHandler{showtimeService: showtimeService}
}

// Create godoc
//
//	@Summary		Schedule a showtime in a hall
//	@Tags			showtimes
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.ShowtimeRequest	true	"Showtime details"
//	@Success		201		{object}	response.Body{data=dto.ShowtimeResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/showtimes [post]
func (h *ShowtimeHandler) Create(c *gin.Context) {
	var req dto.ShowtimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	showtime, err := h.showtimeService.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, showtime)
}

// Update godoc
//
//	@Summary		Reschedule a showtime
//	@Tags			showtimes
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string				true	"Showtime ID"
//	@Param			payload	body		dto.ShowtimeRequest	true	"Showtime details"
//	@Success		200		{object}	response.Body{data=dto.ShowtimeResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/showtimes/{id} [put]
func (h *ShowtimeHandler) Update(c *gin.Context) {
	var req dto.ShowtimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	showtime, err := h.showtimeService.Update(ctx, c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, showtime)
}

// Delete godoc
//
//	@Summary		Delete a showtime
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Showtime ID"
//	@Success		200	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/admin/showtimes/{id} [delete]
func (h *ShowtimeHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.showtimeService.Delete(ctx, c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "deleted")
}

// ListForMovie godoc
//
//	@Summary		List showtimes of a movie for a date
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	false	"Movie ID"
//	@Param			date	query		string	false	"Date (YYYY-MM-DD)"
//	@Success		200		{object}	response.Body{data=[]dto.ShowtimeListItem}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Router			/movies/{id}/showtimes [get]
func (h *ShowtimeHandler) ListForMovie(c *gin.Context) {
	items, err := h.showtimeService.ListByMovie(c.Request.Context(), c.Param("id"), c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, items)
}

// List godoc
//
//	@Summary		Showtimes on sale for a date, all movies
//	@Description	Hides draft/ended movies, started showtimes and halls without full prices. A day without showtimes returns an empty list.
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			date	query		string	false	"Date (YYYY-MM-DD), default today"
//	@Success		200		{object}	response.Body{data=[]dto.ShowtimeListItem}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Router			/showtimes [get]
func (h *ShowtimeHandler) List(c *gin.Context) {
	items, err := h.showtimeService.ListByDate(c.Request.Context(), c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, items)
}

// SeatMap godoc
//
//	@Summary		Get the seat grid of a showtime with prices
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Showtime ID"
//	@Success		200	{object}	response.Body{data=dto.SeatMapResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/shows/{id}/seats [get]
func (h *ShowtimeHandler) SeatMap(c *gin.Context) {
	seatMap, err := h.showtimeService.SeatMap(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seatMap)
}
