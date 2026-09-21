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

//	@Summary		Open an empty booking for a showtime
//	@Description	Creates a seatless PENDING booking so the client can count down from entry; seats attach later with POST /orders/hold (replace path). Re-entering with a still-valid pending booking returns it.
//	@Tags			orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.InitRequest	true	"Showtime to open a booking for"
//	@Success		201		{object}	response.Body{data=dto.InitResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/orders/init [post]
func (h *BookingHandler) Init(c *gin.Context) {
	var req dto.InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.bookingService.Init(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}

//	@Summary		Extend a pending booking for the heartbeat
//	@Description	Silent heartbeat: pushes expiry forward by one TTL, never past created_at + booking.hold_max_lifetime_minutes. Refused for paid, closed or seat-lost bookings.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Booking ID"
//	@Success		200		{object}	response.Body{data=dto.RefreshResponse}
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/orders/{id}/refresh [post]
func (h *BookingHandler) Refresh(c *gin.Context) {
	res, err := h.bookingService.Refresh(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

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

//	@Summary		List the current user's payment transactions
//	@Description	Financial history (amount, provider, status, refunds) with just enough booking/showtime context to place each one — distinct from GET /orders, which is the booking/ticket history.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page"	default(1)
//	@Param			page_size	query	int	false	"Page size"	default(10)
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.TransactionResponse}}
//	@Failure		401			{object}	response.Body
//	@Router			/users/me/transactions [get]
func (h *BookingHandler) Transactions(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	items, total, err := h.bookingService.Transactions(c.Request.Context(), middleware.CurrentUserID(c), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, items, q.Page, q.PageSize, total)
}

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

//	@Summary		Get a ticket's QR code
//	@Description	Base64 PNG, same encoding the ticket email embeds inline. Works whether or not the showtime has started; only the ticket's own buyer or staff/admin may fetch it.
//	@Tags			tickets
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Ticket ID"
//	@Success		200	{object}	response.Body{data=dto.TicketQRResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/tickets/{id}/qr [get]
func (h *BookingHandler) TicketQR(c *gin.Context) {
	res, err := h.bookingService.TicketQR(c.Request.Context(),
		middleware.CurrentUserID(c), middleware.CurrentUserRole(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

//	@Summary		List every order for operators, paged and filterable
//	@Description	The operator counterpart of GET /orders: not scoped to the caller. Online and counter sales alike, with the buyer's identity, the showtime and the payment attempt the order carries. It reports what the database holds and does not reconcile with the provider — use GET /staff/orders/{id} for that.
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page			query		int		false	"Page"			default(1)
//	@Param			page_size		query		int		false	"Page size"		default(10)
//	@Param			search			query		string	false	"Booking id, customer email/name/phone, or movie title"
//	@Param			status			query		string	false	"pending | confirmed | expired | refunded"
//	@Param			payment_status	query		string	false	"pending | paid | failed | refund_pending | refunded (matches only an order that already carries a payment)"
//	@Param			sold_via		query		string	false	"online | counter"
//	@Param			showtime_id		query		string	false	"Showtime ID"
//	@Param			movie_id		query		string	false	"Movie ID"
//	@Param			user_id			query		string	false	"Customer ID"
//	@Param			date			query		string	false	"Single day of created_at (YYYY-MM-DD), wins over from/to"
//	@Param			from			query		string	false	"created_at range start (YYYY-MM-DD)"
//	@Param			to				query		string	false	"created_at range end, inclusive (YYYY-MM-DD)"
//	@Param			sort			query		string	false	"created_at | paid_at | total_amount | start_at"
//	@Param			order			query		string	false	"asc or desc"
//	@Success		200				{object}	response.Body{data=response.Paged{items=[]dto.AdminOrderListItem}}
//	@Failure		400				{object}	response.Body
//	@Failure		401				{object}	response.Body
//	@Failure		403				{object}	response.Body
//	@Router			/admin/orders [get]
func (h *BookingHandler) AdminList(c *gin.Context) {
	var query dto.AdminOrderListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	items, total, err := h.bookingService.AdminList(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, items, query.Page, query.PageSize, total)
}
