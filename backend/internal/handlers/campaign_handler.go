package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type CampaignHandler struct {
	campaignService service.CampaignService
}

func NewCampaignHandler(campaignService service.CampaignService) *CampaignHandler {
	return &CampaignHandler{campaignService: campaignService}
}

// @Summary		List active campaigns (public)
// @Description	Only campaigns with active=true and now inside [starts_at, ends_at). Each carries its linked discount codes (code + remaining, never internal counters), combos (with promo_price if set) and published articles.
// @Tags			campaigns
// @Produce		json
// @Param			page		query		int		false	"Page"		default(1)
// @Param			page_size	query		int		false	"Page size"	default(10)
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.CampaignPublicResponse}}
// @Failure		400			{object}	response.Body
// @Router			/campaigns [get]
func (h *CampaignHandler) PublicList(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	rows, total, err := h.campaignService.PublicList(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, q.Page, q.PageSize, total)
}

// @Summary		Get one active campaign (public)
// @Description	404 when the campaign does not exist, is not active, or its window has not started/has ended.
// @Tags			campaigns
// @Produce		json
// @Param			id	path		string	true	"Campaign id"
// @Success		200	{object}	response.Body{data=dto.CampaignPublicResponse}
// @Failure		404	{object}	response.Body
// @Router			/campaigns/{id} [get]
func (h *CampaignHandler) PublicGet(c *gin.Context) {
	found, err := h.campaignService.PublicGet(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, found)
}

// @Summary		List campaigns (admin)
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			page		query		int		false	"Page"		default(1)
// @Param			page_size	query		int		false	"Page size"	default(10)
// @Param			search		query		string	false	"Match name or description"
// @Param			active		query		bool	false	"Filter enabled/disabled; omit for both"
// @Success		200			{object}	response.Body{data=response.Paged{items=[]dto.CampaignResponse}}
// @Failure		400			{object}	response.Body
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Router			/admin/campaigns [get]
func (h *CampaignHandler) AdminList(c *gin.Context) {
	var q dto.CampaignListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()
	rows, total, err := h.campaignService.AdminList(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, q.Page, q.PageSize, total)
}

// @Summary		Get one campaign with everything attached (admin)
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Campaign id"
// @Success		200	{object}	response.Body{data=dto.CampaignDetailResponse}
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/campaigns/{id} [get]
func (h *CampaignHandler) AdminGet(c *gin.Context) {
	found, err := h.campaignService.AdminGet(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, found)
}

// @Summary		Create a campaign draft (admin)
// @Tags			campaigns
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			payload	body		dto.CreateCampaignRequest	true	"Campaign"
// @Success		201		{object}	response.Body{data=dto.CampaignResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Router			/admin/campaigns [post]
func (h *CampaignHandler) AdminCreate(c *gin.Context) {
	var req dto.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	created, err := h.campaignService.AdminCreate(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, created)
}

// @Summary		Update a campaign (admin)
// @Description	PARTIAL update; an omitted field is left alone.
// @Tags			campaigns
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string						true	"Campaign id"
// @Param			payload	body		dto.UpdateCampaignRequest	true	"Fields to change"
// @Success		200		{object}	response.Body{data=dto.CampaignResponse}
// @Failure		400		{object}	response.Body
// @Failure		401		{object}	response.Body
// @Failure		403		{object}	response.Body
// @Failure		404		{object}	response.Body
// @Router			/admin/campaigns/{id} [patch]
func (h *CampaignHandler) AdminUpdate(c *gin.Context) {
	var req dto.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	updated, err := h.campaignService.AdminUpdate(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, updated)
}

// @Summary		Delete a campaign (admin)
// @Description	Hard delete: its combo/article links go with it, and any discount code it had gets campaign_id set back to NULL (never orphaned, never deleted).
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id	path	string	true	"Campaign id"
// @Success		204	"No Content"
// @Failure		401	{object}	response.Body
// @Failure		403	{object}	response.Body
// @Failure		404	{object}	response.Body
// @Router			/admin/campaigns/{id} [delete]
func (h *CampaignHandler) AdminDelete(c *gin.Context) {
	if err := h.campaignService.AdminDelete(c.Request.Context(), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary		Attach a combo to a campaign (admin)
// @Description	Upserts: attaching an already-linked combo updates its promo_price. v1 record/display only - it does not change what /combo-orders charges.
// @Tags			campaigns
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id			path		string						true	"Campaign id"
// @Param			combo_id	path		string						true	"Combo id"
// @Param			payload		body		dto.AttachComboRequest		false	"Optional promo price"
// @Success		200			{object}	response.Body{data=dto.CampaignComboResponse}
// @Failure		400			{object}	response.Body
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/combos/{combo_id} [post]
func (h *CampaignHandler) AttachCombo(c *gin.Context) {
	// The body is optional: promo_price alone is nice-to-have, so an empty
	// request (no promo price at all) is not a binding error.
	var req dto.AttachComboRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, err)
		return
	}
	linked, err := h.campaignService.AttachCombo(c.Request.Context(), c.Param("id"), c.Param("combo_id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, linked)
}

// @Summary		Detach a combo from a campaign (admin)
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id			path	string	true	"Campaign id"
// @Param			combo_id	path	string	true	"Combo id"
// @Success		204			"No Content"
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/combos/{combo_id} [delete]
func (h *CampaignHandler) DetachCombo(c *gin.Context) {
	if err := h.campaignService.DetachCombo(c.Request.Context(), c.Param("id"), c.Param("combo_id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary		Attach an article to a campaign (admin)
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id			path		string	true	"Campaign id"
// @Param			article_id	path		string	true	"Article id"
// @Success		200			{object}	response.Body{data=dto.ArticleResponse}
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/articles/{article_id} [post]
func (h *CampaignHandler) AttachArticle(c *gin.Context) {
	linked, err := h.campaignService.AttachArticle(c.Request.Context(), c.Param("id"), c.Param("article_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, linked)
}

// @Summary		Detach an article from a campaign (admin)
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id			path	string	true	"Campaign id"
// @Param			article_id	path	string	true	"Article id"
// @Success		204			"No Content"
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/articles/{article_id} [delete]
func (h *CampaignHandler) DetachArticle(c *gin.Context) {
	if err := h.campaignService.DetachArticle(c.Request.Context(), c.Param("id"), c.Param("article_id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// @Summary		Attach a discount code to a campaign (admin)
// @Description	A code belongs to at most one campaign; attaching it here sets discount_codes.campaign_id. Applying it then additionally requires the campaign to be active and inside its own window.
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id			path		string	true	"Campaign id"
// @Param			code_id		path		string	true	"Discount code id"
// @Success		200			{object}	response.Body{data=dto.DiscountCodeResponse}
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/discount-codes/{code_id} [post]
func (h *CampaignHandler) AttachDiscountCode(c *gin.Context) {
	linked, err := h.campaignService.AttachDiscountCode(c.Request.Context(), c.Param("id"), c.Param("code_id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, linked)
}

// @Summary		Detach a discount code from a campaign (admin)
// @Description	Sets discount_codes.campaign_id back to NULL; the code itself is untouched and stays usable as a standalone code.
// @Tags			campaigns
// @Produce		json
// @Security		BearerAuth
// @Param			id			path	string	true	"Campaign id"
// @Param			code_id		path	string	true	"Discount code id"
// @Success		204			"No Content"
// @Failure		401			{object}	response.Body
// @Failure		403			{object}	response.Body
// @Failure		404			{object}	response.Body
// @Router			/admin/campaigns/{id}/discount-codes/{code_id} [delete]
func (h *CampaignHandler) DetachDiscountCode(c *gin.Context) {
	if err := h.campaignService.DetachDiscountCode(c.Request.Context(), c.Param("id"), c.Param("code_id")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
