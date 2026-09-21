package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ComboHandler struct {
	comboService service.ComboService
}

func NewComboHandler(comboService service.ComboService) *ComboHandler {
	return &ComboHandler{comboService: comboService}
}

//	@Summary		List active combo/concession products
//	@Tags			combos
//	@Produce		json
//	@Success		200	{object}	response.Body{data=[]dto.ComboResponse}
//	@Router			/combos [get]
func (h *ComboHandler) List(c *gin.Context) {
	combos, err := h.comboService.ListActive(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combos)
}

//	@Summary		Place a combo/concession order
//	@Description	Independent of the ticket booking flow; booking_id is an optional correlation only.
//	@Tags			combos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.CreateComboOrderRequest	true	"Combo order"
//	@Success		201		{object}	response.Body{data=dto.ComboOrderResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		404		{object}	response.Body	"booking_id does not belong to the caller"
//	@Router			/combo-orders [post]
func (h *ComboHandler) CreateOrder(c *gin.Context) {
	var req dto.CreateComboOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	order, err := h.comboService.CreateOrder(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, order)
}

//	@Summary		List the current user's combo orders
//	@Tags			combos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int	false	"Page"	default(1)
//	@Param			page_size	query	int	false	"Page size"	default(10)
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ComboOrderResponse}}
//	@Failure		401			{object}	response.Body
//	@Router			/combo-orders/me [get]
func (h *ComboHandler) ListMyOrders(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	orders, total, err := h.comboService.ListMyOrders(c.Request.Context(), middleware.CurrentUserID(c), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, orders, q.Page, q.PageSize, total)
}
