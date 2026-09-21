package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type HallHandler struct {
	hallService service.HallService
}

func NewHallHandler(hallService service.HallService) *HallHandler {
	return &HallHandler{hallService: hallService}
}

//	@Summary		List halls (paginated)
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			search		query		string	false	"Search by hall name"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.HallResponse}}
//	@Failure		401			{object}	response.Body
//	@Router			/admin/halls [get]
func (h *HallHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()

	halls, total, err := h.hallService.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, halls, query.Page, query.PageSize, total)
}

//	@Summary		Get hall details
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Hall ID"
//	@Success		200	{object}	response.Body{data=dto.HallResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/admin/halls/{id} [get]
func (h *HallHandler) Show(c *gin.Context) {
	hall, err := h.hallService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hall)
}

//	@Summary		Create a hall and generate its seat grid
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.HallRequest	true	"Hall layout"
//	@Success		201		{object}	response.Body{data=dto.HallResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls [post]
func (h *HallHandler) Create(c *gin.Context) {
	var req dto.HallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	hall, err := h.hallService.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hall)
}

//	@Summary		Get the seat grid of a hall
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Hall ID"
//	@Success		200	{object}	response.Body{data=[]dto.SeatResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/admin/halls/{id}/seats [get]
func (h *HallHandler) Seats(c *gin.Context) {
	seats, err := h.hallService.SeatsByHall(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSeatResponses(seats))
}

//	@Summary		Change a seat's type or gap flag
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string					true	"Hall ID"
//	@Param			seatId	path	string					true	"Seat ID"
//	@Param			payload	body	dto.SeatUpdateRequest	true	"Seat changes"
//	@Success		200		{object}	response.Body{data=dto.SeatResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/halls/{id}/seats/{seatId} [put]
func (h *HallHandler) UpdateSeat(c *gin.Context) {
	var req dto.SeatUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	seat, err := h.hallService.UpdateSeat(ctx, c.Param("id"), c.Param("seatId"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seat)
}

//	@Summary		Set prices for all seat types of a hall
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string			true	"Hall ID"
//	@Param			payload	body		dto.PriceRequest	true	"Price per seat type"
//	@Success		200		{object}	response.Body{data=[]dto.HallPriceResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/halls/{id}/prices [put]
func (h *HallHandler) SetPrices(c *gin.Context) {
	var req dto.PriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	prices, err := h.hallService.SetPrices(ctx, c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, prices)
}

//	@Summary		Preview the built-in hall templates
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.HallTemplateResponse}
//	@Failure		401	{object}	response.Body
//	@Router			/admin/hall-templates [get]
func (h *HallHandler) Templates(c *gin.Context) {
	response.OK(c, h.hallService.Templates())
}

//	@Summary		Clone a hall's current seat grid under a new name
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Hall ID to clone"
//	@Param			payload	body		dto.CloneHallRequest	true	"New hall name"
//	@Success		201		{object}	response.Body{data=dto.HallResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id}/clone [post]
func (h *HallHandler) Clone(c *gin.Context) {
	var req dto.CloneHallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	hall, err := h.hallService.Clone(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hall)
}

//	@Summary		Change many seats at once (labels, rows, columns or a range)
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"Hall ID"
//	@Param			payload	body		dto.BulkSeatUpdateRequest	true	"Changes, one selector each"
//	@Success		200		{object}	response.Body{data=[]dto.SeatResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id}/seats [patch]
func (h *HallHandler) BulkUpdateSeats(c *gin.Context) {
	var req dto.BulkSeatUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	seats, err := h.hallService.BulkUpdateSeats(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seats)
}

//	@Summary		Rename a hall, change its screen/aisle display, or (de)activate it
//	@Description	Deactivating is refused (409) while an open showtime is still to come; once inactive the hall takes no new showtimes.
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Hall ID"
//	@Param			payload	body		dto.UpdateHallRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.HallResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id} [put]
func (h *HallHandler) UpdateHall(c *gin.Context) {
	var req dto.UpdateHallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	hall, err := h.hallService.UpdateHall(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hall)
}

//	@Summary		Regenerate a hall's whole seat grid
//	@Description	Only while the hall has never had a single booking (409 otherwise).
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string			true	"Hall ID"
//	@Param			payload	body		dto.HallRequest	true	"New layout (name/prices are ignored here)"
//	@Success		200		{object}	response.Body{data=dto.HallResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id}/layout [put]
func (h *HallHandler) RegenerateLayout(c *gin.Context) {
	var req dto.HallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	hall, err := h.hallService.RegenerateLayout(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hall)
}

//	@Summary		Append one new row of standard seats to a hall
//	@Description	Never touches an existing seat - safe while the hall has a live booking (409 only on an actual conflict), unlike regenerating the whole layout.
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Hall ID"
//	@Success		201	{object}	response.Body{data=[]dto.SeatResponse}
//	@Failure		400	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/admin/halls/{id}/seats/rows [post]
func (h *HallHandler) AddRow(c *gin.Context) {
	seats, err := h.hallService.AddRow(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, seats)
}

//	@Summary		Remove one row and renumber every row after it
//	@Description	Any row can be removed, not just the last - rows after it shift down by one so numbering stays 1..N with no gap. Refused (409) if any seat in the removed row has booking history.
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"Hall ID"
//	@Param			rowLabel	path		string	true	"Row label, e.g. C"
//	@Success		200			{object}	response.Body{data=[]dto.SeatResponse}
//	@Failure		400			{object}	response.Body
//	@Failure		401			{object}	response.Body
//	@Failure		404			{object}	response.Body
//	@Failure		409			{object}	response.Body
//	@Router			/admin/halls/{id}/seats/rows/{rowLabel} [delete]
func (h *HallHandler) DeleteRow(c *gin.Context) {
	seats, err := h.hallService.DeleteRow(c.Request.Context(), c.Param("id"), c.Param("rowLabel"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seats)
}

//	@Summary		Merge two adjacent standard seats into one couple seat
//	@Description	The right seat is deleted entirely; the left seat becomes col_span=2, seat_type=couple. Refused (409) if either seat has booking history.
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Hall ID"
//	@Param			payload	body		dto.MergeSeatsRequest	true	"Left/right seat labels"
//	@Success		200		{object}	response.Body{data=dto.SeatResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id}/seats/merge [post]
func (h *HallHandler) MergeSeats(c *gin.Context) {
	var req dto.MergeSeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	seat, err := h.hallService.MergeSeats(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seat)
}

//	@Summary		Split one couple seat back into two standard seats
//	@Description	The seat becomes col_span=1, and a new standard seat is created at the next column. Refused (409) if the seat has booking history.
//	@Tags			halls
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Hall ID"
//	@Param			payload	body		dto.SplitSeatRequest	true	"Couple seat label"
//	@Success		200		{object}	response.Body{data=[]dto.SeatResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/halls/{id}/seats/split [post]
func (h *HallHandler) SplitSeat(c *gin.Context) {
	var req dto.SplitSeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	seats, err := h.hallService.SplitSeat(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, seats)
}

//	@Summary		Delete a hall (soft delete)
//	@Description	Refused (409) while it has a showtime not yet ended.
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Hall ID"
//	@Success		204	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/admin/halls/{id} [delete]
func (h *HallHandler) DeleteHall(c *gin.Context) {
	if err := h.hallService.DeleteHall(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

//	@Summary		Get the prices of a hall
//	@Tags			halls
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Hall ID"
//	@Success		200	{object}	response.Body{data=[]dto.HallPriceResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/admin/halls/{id}/prices [get]
func (h *HallHandler) Prices(c *gin.Context) {
	prices, err := h.hallService.PricesByHall(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	rows := make([]dto.HallPriceResponse, 0, len(prices))
	for _, p := range prices {
		rows = append(rows, dto.HallPriceResponse{SeatType: p.SeatType, Price: p.Price})
	}
	response.OK(c, rows)
}
