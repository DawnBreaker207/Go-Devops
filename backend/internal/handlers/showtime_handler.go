package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ShowtimeHandler struct {
	showtimeService service.ShowtimeService
	bookingService  service.BookingService
}

func NewShowtimeHandler(showtimeService service.ShowtimeService, bookingService service.BookingService) *ShowtimeHandler {
	return &ShowtimeHandler{showtimeService: showtimeService, bookingService: bookingService}
}

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

//	@Summary		Cancel an already-sold showtime and refund its bookings
//	@Description	Unlike DELETE (refused once a showtime has any booking), this cancels the showtime and refunds every booking that already paid for it through the normal refund pipeline. Voids issued tickets of confirmed bookings.
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Showtime ID"
//	@Success		200	{object}	response.Body{data=dto.ShowtimeCancelResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body	"showtime is already cancelled"
//	@Router			/admin/showtimes/{id}/cancel [post]
func (h *ShowtimeHandler) Cancel(c *gin.Context) {
	result, err := h.bookingService.CancelShowtime(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result)
}

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

//	@Summary		List showtimes for operators, paged and filterable
//	@Description	Shows everything the public picker hides: closed showtimes, draft or ended movies, past dates and halls without a full price set.
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page"	default(1)
//	@Param			page_size	query		int		false	"Page size"	default(10)
//	@Param			search		query		string	false	"Match movie title or hall name"
//	@Param			movie_id	query		string	false	"Movie ID"
//	@Param			hall_id		query		string	false	"Hall ID"
//	@Param			status		query		string	false	"open or closed"
//	@Param			date		query		string	false	"Single day (YYYY-MM-DD), wins over from/to"
//	@Param			from		query		string	false	"Range start (YYYY-MM-DD)"
//	@Param			to			query		string	false	"Range end, inclusive (YYYY-MM-DD)"
//	@Param			sort		query		string	false	"start_at or created_at"
//	@Param			order		query		string	false	"asc or desc"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ShowtimeResponse}}
//	@Failure		400			{object}	response.Body
//	@Failure		401			{object}	response.Body
//	@Failure		403			{object}	response.Body
//	@Router			/admin/showtimes [get]
func (h *ShowtimeHandler) AdminList(c *gin.Context) {
	var query dto.ShowtimeAdminListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()

	items, total, err := h.showtimeService.AdminList(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, items, query.Page, query.PageSize, total)
}

//	@Summary		Get one showtime for operators
//	@Description	Unlike the booking endpoints this does not require the showtime to be on sale.
//	@Tags			showtimes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Showtime ID"
//	@Success		200	{object}	response.Body{data=dto.ShowtimeResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/admin/showtimes/{id} [get]
func (h *ShowtimeHandler) Detail(c *gin.Context) {
	showtime, err := h.showtimeService.Detail(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, showtime)
}

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
