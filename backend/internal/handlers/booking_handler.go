package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type BookingHandler struct {
	bookingService service.BookingService
}

func NewBookingHandler(bookingService service.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bookingService}
}

// Hold godoc
//
//	@Summary		Hold seats for a showtime
//	@Description	Reserves seats for 10 minutes and locks their price. A second hold of the same user/show replaces the first without extending its expiry.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.HoldRequest	true	"Seats to hold"
//	@Success		201		{object}	response.Body{data=dto.HoldResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/orders/hold [post]
func (h *BookingHandler) Hold(c *gin.Context) {
	var req dto.HoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.bookingService.Hold(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}

// Pay godoc
//
//	@Summary		Start payment for a booking with a chosen provider
//	@Description	Returns the provider checkout URL. The amount comes from the booking. Paying again with the same provider returns the open checkout; another provider opens a new attempt.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string			true	"Booking ID"
//	@Param			payload	body	dto.PayRequest	true	"Provider from GET /payments/providers (empty = default)"
//	@Success		200		{object}	response.Body{data=dto.PayResponse}
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		502		{object}	response.Body
//	@Router			/orders/{id}/pay [post]
func (h *BookingHandler) Pay(c *gin.Context) {
	var req dto.PayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ClientIP = c.ClientIP()
	res, err := h.bookingService.Pay(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// Confirm godoc
//
//	@Summary		Settle a paid booking
//	@Description	Reconciles with the provider, then confirms (issues tickets) or reports the refund. Idempotent.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Booking ID"
//	@Success		200	{object}	response.Body{data=dto.OrderDetailResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/orders/{id}/confirm [post]
func (h *BookingHandler) Confirm(c *gin.Context) {
	res, err := h.bookingService.Confirm(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// Status godoc
//
//	@Summary		Get booking status (reconciles with the provider)
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Booking ID"
//	@Success		200	{object}	response.Body{data=dto.OrderStatusResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/orders/{id}/status [get]
func (h *BookingHandler) Status(c *gin.Context) {
	res, err := h.bookingService.Status(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// Cancel godoc
//
//	@Summary		Release an unpaid hold now
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Booking ID"
//	@Success		200	{object}	response.Body{data=dto.OrderStatusResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/orders/{id}/cancel [post]
func (h *BookingHandler) Cancel(c *gin.Context) {
	res, err := h.bookingService.Cancel(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// List godoc
//
//	@Summary		List the current user's orders
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page"	default(1)
//	@Param			page_size	query	int	false	"Page size"	default(10)
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.OrderStatusResponse}}
//	@Failure		401			{object}	response.Body
//	@Router			/orders [get]
func (h *BookingHandler) List(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	items, total, err := h.bookingService.List(c.Request.Context(), middleware.CurrentUserID(c), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, items, q.Page, q.PageSize, total)
}

// Order godoc
//
//	@Summary		Get the e-ticket of one order (reconciled, with ticket codes)
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Booking ID"
//	@Success		200	{object}	response.Body{data=dto.OrderDetailResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/orders/{id} [get]
func (h *BookingHandler) Order(c *gin.Context) {
	res, err := h.bookingService.Order(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// Redeem godoc
//
//	@Summary		Check a ticket in at the gate (staff)
//	@Description	id is the ticket id or the code read from the QR. Verdict: ok | used | wrong_show | not_found | too_early | closed (outside the check-in window, returned as checkin_opens_at / checkin_closes_at).
//	@Tags			tickets
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string					true	"Ticket ID or QR code"
//	@Param			payload	body	dto.RedeemRequestBody	true	"Showtime at the gate"
//	@Success		200		{object}	response.Body{data=dto.RedeemResponse}
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Router			/tickets/{id}/redeem [post]
func (h *BookingHandler) Redeem(c *gin.Context) {
	var req dto.RedeemRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.bookingService.Redeem(c.Request.Context(), c.Param("id"), req.ShowtimeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
