package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type PricingHandler struct {
	pricingService service.PricingService
}

func NewPricingHandler(pricingService service.PricingService) *PricingHandler {
	return &PricingHandler{pricingService: pricingService}
}

// @Summary		Get the global base prices (admin)
// @Description	One row per seat type. All four seat types always exist once migration 000013 has run.
// @Tags			pricing
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	response.Body{data=[]dto.BasePriceResponse}
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Router			/admin/pricing/base [get]
func (h *PricingHandler) AdminGetBasePrices(c *gin.Context) {
	prices, err := h.pricingService.AdminGetBasePrices(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, prices)
}

// @Summary		Set the global base price for one or more seat types (admin)
// @Description	Send just the seat type(s) you are changing; the rest are left alone.
// @Tags			pricing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.BasePriceRequest	true	"Seat type -> price"
// @Success		200		{object}	response.Body{data=[]dto.BasePriceResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Router			/admin/pricing/base [put]
func (h *PricingHandler) AdminSetBasePrices(c *gin.Context) {
	var req dto.BasePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	prices, err := h.pricingService.AdminSetBasePrices(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, prices)
}

// @Summary		List pricing rules (admin)
// @Tags			pricing
// @Produce		json
// @Security		BearerAuth
// @Param			page		query		int		false	"Page"		default(1)
// @Param			page_size	query		int		false	"Page size"	default(10)
// @Param			search		query		string	false	"Match rule name"
// @Param			active		query		bool	false	"Filter enabled/disabled; omit for both"
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.PricingRuleResponse}}
// @Failure		400			{object}	response.Body
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Router			/admin/pricing/rules [get]
func (h *PricingHandler) AdminListRules(c *gin.Context) {
	var q dto.PricingRuleListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	rules, total, err := h.pricingService.AdminListRules(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rules, q.Page, q.PageSize, total)
}

// @Summary		Get one pricing rule (admin)
// @Tags			pricing
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Pricing rule id"
// @Success		200	{object}	response.Body{data=dto.PricingRuleResponse}
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/pricing/rules/{id} [get]
func (h *PricingHandler) AdminGetRule(c *gin.Context) {
	found, err := h.pricingService.AdminGetRule(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, found)
}

// @Summary		Create a pricing rule (admin)
// @Tags			pricing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.CreatePricingRuleRequest	true	"Rule"
// @Success		201		{object}	response.Body{data=dto.PricingRuleResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Router			/admin/pricing/rules [post]
func (h *PricingHandler) AdminCreateRule(c *gin.Context) {
	var req dto.CreatePricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.pricingService.AdminCreateRule(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// @Summary		Update a pricing rule (admin)
// @Description	PARTIAL update. Sending "" for start_time/end_time/specific_date clears it back to NULL.
// @Tags			pricing
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string							true	"Pricing rule id"
// @Param			payload	body		dto.UpdatePricingRuleRequest	true	"Fields to change"
// @Success		200		{object}	response.Body{data=dto.PricingRuleResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Failure		404		{object}	response.Body
// @Router			/admin/pricing/rules/{id} [patch]
func (h *PricingHandler) AdminUpdateRule(c *gin.Context) {
	var req dto.UpdatePricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.pricingService.AdminUpdateRule(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, updated)
}

// @Summary		Delete a pricing rule (admin)
// @Tags			pricing
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Pricing rule id"
// @Success		204	"No Content"
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/pricing/rules/{id} [delete]
func (h *PricingHandler) AdminDeleteRule(c *gin.Context) {
	if err := h.pricingService.AdminDeleteRule(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary		Public ticket price list
// @Description	The global base price for each seat type, plus the cheapest configured seat anywhere. No auth: this is the customer price page. Replaces the old per-hall price list (PLAN_CAMPAIGN.md section 11.4, Phase 3) now that hall_prices is gone - every hall shares the same prices.
// @Tags			pricing
// @Produce		json
// @Success		200	{object}	response.Body{data=dto.PublicPriceListResponse}
// @Failure		500	{object}	response.Body
// @Router			/pricing [get]
func (h *PricingHandler) PublicPrices(c *gin.Context) {
	prices, err := h.pricingService.PublicPrices(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, prices)
}

// @Summary		Preview the price for one showtime and seat type
// @Description	Read-only: base price, every rule that matched (in applied order) and the floored-at-0 final price. Never touches booking or hold state.
// @Tags			pricing
// @Produce		json
// @Param			showtime_id	query		string	true	"Showtime id"
// @Param			seat_type	query		string	true	"standard, vip, couple or recliner"
// @Success		200			{object}	response.Body{data=dto.PricingQuoteResponse}
// @Failure		400			{object}	response.Body
// @Failure		404			{object}	response.Body	"showtime not found"
// @Router			/pricing/quote [get]
func (h *PricingHandler) Quote(c *gin.Context) {
	var q dto.PricingQuoteQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	result, err := h.pricingService.Quote(c.Request.Context(), q.ShowtimeID, q.SeatType)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result)
}
