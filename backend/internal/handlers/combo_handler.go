package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ComboHandler struct {
	comboService service.ComboService
}

func NewComboHandler(comboService service.ComboService) *ComboHandler {
	return &ComboHandler{comboService: comboService}
}

// List godoc
//
//	@Summary		List combos
//	@Tags			combos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			active_only	query		bool	false	"Only active combos"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.ComboResponse}}
//	@Router			/admin/combos [get]
func (h *ComboHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	activeOnly := c.Query("active_only") == "true"
	combos, total, err := h.comboService.List(c.Request.Context(), activeOnly, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, combos, query.Page, query.PageSize, total)
}

// ListPublic godoc
//
//	@Summary		List active combos (public)
//	@Tags			combos
//	@Produce		json
//	@Success		200	{object}	response.Body{data=[]dto.ComboResponse}
//	@Router			/combos [get]
func (h *ComboHandler) ListPublic(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	combos, _, err := h.comboService.List(c.Request.Context(), true, query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combos)
}

// Show godoc
//
//	@Summary		Get a combo
//	@Tags			combos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Combo ID"
//	@Success		200	{object}	response.Body{data=dto.ComboResponse}
//	@Failure		404	{object}	response.Body
//	@Router			/admin/combos/{id} [get]
func (h *ComboHandler) Show(c *gin.Context) {
	combo, err := h.comboService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combo)
}

// Create godoc
//
//	@Summary		Create a combo
//	@Tags			combos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.ComboRequest	true	"Combo"
//	@Success		201		{object}	response.Body{data=dto.ComboResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/combos [post]
func (h *ComboHandler) Create(c *gin.Context) {
	var req dto.ComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	combo, err := h.comboService.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, combo)
}

// Update godoc
//
//	@Summary		Update a combo
//	@Tags			combos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Combo ID"
//	@Param			payload	body		dto.UpdateComboRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.ComboResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/combos/{id} [put]
func (h *ComboHandler) Update(c *gin.Context) {
	var req dto.UpdateComboRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	combo, err := h.comboService.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, combo)
}

// SetStock godoc
//
//	@Summary		Cap a combo's stock at one branch (NULL/no row = unlimited)
//	@Tags			combos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string					true	"Combo ID"
//	@Param			payload	body	dto.SetComboStockRequest	true	"Branch and quantity"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/combos/{id}/stock [put]
func (h *ComboHandler) SetStock(c *gin.Context) {
	var req dto.SetComboStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.comboService.SetStock(c.Request.Context(), req.BranchID, c.Param("id"), req.StockQuantity); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "updated")
}
