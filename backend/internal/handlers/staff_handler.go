package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type StaffHandler struct {
	reports  service.ReportService
	bookings service.BookingService
}

func NewStaffHandler(reports service.ReportService, bookings service.BookingService) *StaffHandler {
	return &StaffHandler{reports: reports, bookings: bookings}
}

// Dashboard godoc
//
//	@Summary		Staff board: showtimes of a day with seats sold / held / checked in
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			date	query		string	false	"Local date YYYY-MM-DD (default today)"
//	@Success		200		{object}	response.Body{data=dto.StaffBoardResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Router			/staff/dashboard [get]
func (h *StaffHandler) Dashboard(c *gin.Context) {
	board, err := h.reports.StaffBoard(c.Request.Context(), c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, board)
}

// Tickets godoc
//
//	@Summary		Tickets of a showtime (status=issued: still waiting at the gate)
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string	true	"Showtime ID"
//	@Param			status	query		string	false	"issued | redeemed"
//	@Success		200		{object}	response.Body{data=[]dto.StaffTicketResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/staff/showtimes/{id}/tickets [get]
func (h *StaffHandler) Tickets(c *gin.Context) {
	tickets, err := h.reports.ShowtimeTickets(c.Request.Context(), c.Param("id"), c.Query("status"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tickets)
}

// CounterSell godoc
//
//	@Summary		Walk-in sale at the counter
//	@Description	Sells tickets for cash to someone without an account: booking is confirmed immediately, no email.
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.CounterSellRequest	true	"Showtime, seats and walk-in details"
//	@Success		200		{object}	response.Body{data=dto.OrderDetailResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		409		{object}	response.Body	"seat already taken"
//	@Router			/staff/orders [post]
func (h *StaffHandler) CounterSell(c *gin.Context) {
	var req dto.CounterSellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	order, err := h.bookings.CounterSell(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, order)
}

// BoxOfficeDay godoc
//
//	@Summary		Counter sales of a day (close-day report)
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			date	query	string	false	"YYYY-MM-DD, default today"
//	@Success		200		{object}	response.Body{data=dto.BoxOfficeDayResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/staff/boxoffice/day [get]
func (h *StaffHandler) BoxOfficeDay(c *gin.Context) {
	day, err := h.reports.BoxOfficeDay(c.Request.Context(), c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, day)
}
