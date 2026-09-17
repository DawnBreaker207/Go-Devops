package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type PricingRuleHandler struct {
	rules service.PricingRuleService
}

func NewPricingRuleHandler(rules service.PricingRuleService) *PricingRuleHandler {
	return &PricingRuleHandler{rules: rules}
}

// List godoc
//
//	@Summary		List dynamic pricing rules
//	@Tags			pricing-rules
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.PricingRuleResponse}
//	@Router			/admin/pricing-rules [get]
func (h *PricingRuleHandler) List(c *gin.Context) {
	rows, err := h.rules.List(c.Request.Context(), false)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// Create godoc
//
//	@Summary		Create a dynamic pricing rule
//	@Tags			pricing-rules
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.PricingRuleRequest	true	"Rule"
//	@Success		201		{object}	response.Body{data=dto.PricingRuleResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/pricing-rules [post]
func (h *PricingRuleHandler) Create(c *gin.Context) {
	var req dto.PricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	rule, err := h.rules.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, rule)
}

// Update godoc
//
//	@Summary		Update a dynamic pricing rule
//	@Tags			pricing-rules
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Rule ID"
//	@Param			payload	body		dto.UpdatePricingRuleRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.PricingRuleResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/pricing-rules/{id} [put]
func (h *PricingRuleHandler) Update(c *gin.Context) {
	var req dto.UpdatePricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	rule, err := h.rules.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rule)
}
