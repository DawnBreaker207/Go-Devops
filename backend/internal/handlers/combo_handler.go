package handlers

import (
	"net/http"

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

// @Summary		List active combo/concession products
// @Tags			combos
// @Produce		json
// @Success		200	{object}	response.Body{data=[]dto.ComboResponse}
// @Router			/combos [get]
func (h *ComboHandler) List(c *gin.Context) {
	combos, err := h.comboService.ListActive(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combos)
}

// @Summary		Place a combo/concession order
// @Description	Independent of the ticket booking flow; booking_id is an optional correlation only.
// @Tags			combos
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.CreateComboOrderRequest	true	"Combo order"
// @Success		201		{object}	response.Body{data=dto.ComboOrderResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		404		{object}	response.Body	"booking_id does not belong to the caller"
// @Router			/combo-orders [post]
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

// @Summary		List the current user's combo orders
// @Tags			combos
// @Produce		json
// @Security		BearerAuth
// @Param			page		query	int	false	"Page"	default(1)
// @Param			page_size	query	int	false	"Page size"	default(10)
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ComboOrderResponse}}
// @Failure		401			{object}	response.Body
// @Router			/combo-orders/me [get]
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

/* -------------------------------------------------------------------------- */
/* Operator catalogue: /admin/concessions (admin AND staff, like halls)        */
/* -------------------------------------------------------------------------- */

// @Summary		List the concession catalogue (operator)
// @Description	Unlike the public GET /combos this is paged and INCLUDES inactive products, so an operator can put one back on sale.
// @Tags			combos
// @Produce		json
// @Security		BearerAuth
// @Param			page		query		int		false	"Page"		default(1)
// @Param			page_size	query		int		false	"Page size"	default(10)
// @Param			search		query		string	false	"Match name or description"
// @Param			active		query		bool	false	"Filter on sale / off sale; omit for both"
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ComboResponse}}
// @Failure		400			{object}	response.Body
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Router			/admin/concessions [get]
func (h *ComboHandler) AdminList(c *gin.Context) {
	var q dto.AdminComboListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	combos, total, err := h.comboService.AdminList(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, combos, q.Page, q.PageSize, total)
}

// @Summary		Get one concession product (operator)
// @Tags			combos
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Combo id"
// @Success		200	{object}	response.Body{data=dto.ComboResponse}
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/concessions/{id} [get]
func (h *ComboHandler) AdminGet(c *gin.Context) {
	combo, err := h.comboService.AdminGet(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combo)
}

// @Summary		Add a concession product
// @Tags			combos
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.CreateComboRequest	true	"Product"
// @Success		201		{object}	response.Body{data=dto.ComboResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Router			/admin/concessions [post]
func (h *ComboHandler) AdminCreate(c *gin.Context) {
	var req dto.CreateComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	combo, err := h.comboService.AdminCreate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, combo)
}

// @Summary		Update a concession product
// @Description	PARTIAL update: an omitted field is left alone. A body with no field at all is 400/40001.
// @Tags			combos
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string					true	"Combo id"
// @Param			payload	body		dto.UpdateComboRequest	true	"Fields to change"
// @Success		200		{object}	response.Body{data=dto.ComboResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Failure		404		{object}	response.Body
// @Router			/admin/concessions/{id} [patch]
func (h *ComboHandler) AdminUpdate(c *gin.Context) {
	var req dto.UpdateComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	combo, err := h.comboService.AdminUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combo)
}

// @Summary		Retire a concession product
// @Description	Soft delete: past combo orders still reference it, so the row stays and only leaves the catalogue.
// @Tags			combos
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Combo id"
// @Success		204	"No Content"
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/concessions/{id} [delete]
func (h *ComboHandler) AdminDelete(c *gin.Context) {
	if err := h.comboService.AdminDelete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	// 204 with an EMPTY body, no envelope - same as DELETE /admin/halls/:id and
	// DELETE /users/me. The frontend must not try to unwrap this one.
	c.Status(http.StatusNoContent)
}
