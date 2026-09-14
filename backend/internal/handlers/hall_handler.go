package handlers

import (
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

// List godoc
//
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

// Show godoc
//
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

// Create godoc
//
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

// Seats godoc
//
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

// UpdateSeat godoc
//
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

// SetPrices godoc
//
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

// Prices godoc
//
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
