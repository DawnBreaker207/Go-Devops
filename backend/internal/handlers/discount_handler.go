package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type DiscountHandler struct {
	discountService service.DiscountService
}

func NewDiscountHandler(discountService service.DiscountService) *DiscountHandler {
	return &DiscountHandler{discountService: discountService}
}

// @Summary		Apply a discount code to a pending order
// @Description	Customer only. `subtotal` stays the seat total; `payable` is what the gateway will charge. Every refusal is 400/40001 with a distinct message, and an unknown code answers the same message as a disabled one so the endpoint cannot be used to enumerate codes.
// @Tags			orders
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Booking id"
// @Param			payload	body		dto.ApplyDiscountRequest	true	"Code"
// @Success		200		{object}	response.Body{data=dto.DiscountAppliedResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body	"the order belongs to another user"
// @Failure		404		{object}	response.Body
// @Failure		409		{object}	response.Body	"already discounted, or no longer pending"
// @Router			/orders/{id}/discount [post]
func (h *DiscountHandler) Apply(c *gin.Context) {
	var req dto.ApplyDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	applied, err := h.discountService.Apply(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, applied)
}

// @Summary		Remove the discount code from a pending order
// @Description	Customer only. Gives the code's redemption back.
// @Tags			orders
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Booking id"
// @Success		200	{object}	response.Body{data=dto.DiscountAppliedResponse}
// @Failure		400	{object}	response.Body	"no discount to remove"
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Failure		409	{object}	response.Body	"no longer pending"
// @Router			/orders/{id}/discount [delete]
func (h *DiscountHandler) Remove(c *gin.Context) {
	cleared, err := h.discountService.Remove(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, cleared)
}

/* -------------------------------------------------------------------------- */
/* Operator catalogue: /admin/discounts - ADMIN ONLY                           */
/* -------------------------------------------------------------------------- */

// @Summary		List discount codes (admin)
// @Tags			discounts
// @Produce		json
// @Security		BearerAuth
// @Param			page		query		int		false	"Page"		default(1)
// @Param			page_size	query		int		false	"Page size"	default(10)
// @Param			search		query		string	false	"Match code or description"
// @Param			active		query		bool	false	"Filter enabled/disabled; omit for both"
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.DiscountCodeResponse}}
// @Failure		400			{object}	response.Body
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Router			/admin/discounts [get]
func (h *DiscountHandler) AdminList(c *gin.Context) {
	var q dto.DiscountListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	codes, total, err := h.discountService.AdminList(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, codes, q.Page, q.PageSize, total)
}

// @Summary		Get one discount code (admin)
// @Tags			discounts
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Discount code id"
// @Success		200	{object}	response.Body{data=dto.DiscountCodeResponse}
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/discounts/{id} [get]
func (h *DiscountHandler) AdminGet(c *gin.Context) {
	found, err := h.discountService.AdminGet(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, found)
}

// @Summary		Create a discount code (admin)
// @Tags			discounts
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.CreateDiscountRequest	true	"Code"
// @Success		201		{object}	response.Body{data=dto.DiscountCodeResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Failure		409		{object}	response.Body	"code already exists"
// @Router			/admin/discounts [post]
func (h *DiscountHandler) AdminCreate(c *gin.Context) {
	var req dto.CreateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.discountService.AdminCreate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// @Summary		Update a discount code (admin)
// @Description	PARTIAL update. `code` and `kind` cannot be changed: either would rewrite what the code meant for orders that already used it.
// @Tags			discounts
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Discount code id"
// @Param			payload	body		dto.UpdateDiscountRequest	true	"Fields to change"
// @Success		200		{object}	response.Body{data=dto.DiscountCodeResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Failure		404		{object}	response.Body
// @Router			/admin/discounts/{id} [patch]
func (h *DiscountHandler) AdminUpdate(c *gin.Context) {
	var req dto.UpdateDiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.discountService.AdminUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, updated)
}

// @Summary		Retire a discount code (admin)
// @Description	Soft delete: orders that used it keep their link, and the code string becomes reusable.
// @Tags			discounts
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Discount code id"
// @Success		204	"No Content"
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/discounts/{id} [delete]
func (h *DiscountHandler) AdminDelete(c *gin.Context) {
	if err := h.discountService.AdminDelete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	// 204, empty body, no envelope - like DELETE /admin/halls/:id.
	c.Status(http.StatusNoContent)
}
