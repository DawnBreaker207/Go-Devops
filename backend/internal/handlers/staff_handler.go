package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type StaffHandler struct {
	reports  service.ReportService
	bookings service.BookingService
	users    service.UserService
}

func NewStaffHandler(reports service.ReportService, bookings service.BookingService, users service.UserService) *StaffHandler {
	return &StaffHandler{reports: reports, bookings: bookings, users: users}
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

// Overview godoc
//
//	@Summary		Staff dashboard: today's showtime board + counter sales + tickets awaiting check-in, in one call
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			date	query		string	false	"Local date YYYY-MM-DD (default today)"
//	@Success		200		{object}	response.Body{data=dto.StaffOverviewResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Router			/staff/overview [get]
func (h *StaffHandler) Overview(c *gin.Context) {
	res, err := h.reports.StaffOverview(c.Request.Context(), c.Query("date"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

// SearchCustomers godoc
//
//	@Summary		Search customer accounts (support lookup)
//	@Description	Only role=customer accounts ever show here; staff/admin accounts stay only in /admin/users.
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int		false	"Page"
//	@Param			page_size	query	int		false	"Page size"
//	@Param			search		query	string	false	"Email, name or phone"
//	@Success		200	{object}	response.Body{data=response.Paged{items=[]dto.UserResponse}}
//	@Failure		403	{object}	response.Body
//	@Router			/staff/customers [get]
func (h *StaffHandler) SearchCustomers(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	users, total, err := h.users.List(c.Request.Context(), dto.UserListQuery{PageQuery: q, Role: models.RoleCustomer})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, users, q.Page, q.PageSize, total)
}

// CustomerProfile godoc
//
//	@Summary		A customer's profile (support lookup)
//	@Description	404 if the id isn't a customer account — this route never confirms a staff/admin account exists.
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"User ID"
//	@Success		200	{object}	response.Body{data=dto.UserResponse}
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/staff/customers/{id} [get]
func (h *StaffHandler) CustomerProfile(c *gin.Context) {
	user, err := h.users.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if user.Role != models.RoleCustomer {
		response.Error(c, apperrors.ErrUserNotFound)
		return
	}
	response.OK(c, user)
}

// CustomerOrders godoc
//
//	@Summary		A customer's booking history (support lookup)
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path	string	true	"User ID"
//	@Param			page		query	int		false	"Page"
//	@Param			page_size	query	int		false	"Page size"
//	@Success		200	{object}	response.Body{data=response.Paged{items=[]dto.OrderStatusResponse}}
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/staff/customers/{id}/orders [get]
func (h *StaffHandler) CustomerOrders(c *gin.Context) {
	user, err := h.users.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	if user.Role != models.RoleCustomer {
		response.Error(c, apperrors.ErrUserNotFound)
		return
	}
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	orders, total, err := h.bookings.List(c.Request.Context(), user.ID, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, orders, q.Page, q.PageSize, total)
}

// OrderDetail godoc
//
//	@Summary		Any order's e-ticket detail, by id (support lookup)
//	@Tags			staff
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Booking ID"
//	@Success		200	{object}	response.Body{data=dto.OrderDetailResponse}
//	@Failure		403	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/staff/orders/{id} [get]
func (h *StaffHandler) OrderDetail(c *gin.Context) {
	order, err := h.bookings.AdminOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, order)
}
