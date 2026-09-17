package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type VoucherHandler struct {
	voucherService service.VoucherService
}

func NewVoucherHandler(voucherService service.VoucherService) *VoucherHandler {
	return &VoucherHandler{voucherService: voucherService}
}

// List godoc
//
//	@Summary		List vouchers
//	@Tags			vouchers
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Current page"	default(1)
//	@Param			page_size	query		int		false	"Records per page"	default(10)
//	@Param			status		query		string	false	"pending_approval | active | rejected | disabled"
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.VoucherResponse}}
//	@Router			/admin/vouchers [get]
func (h *VoucherHandler) List(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	vouchers, total, err := h.voucherService.List(c.Request.Context(), c.Query("status"), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, vouchers, query.Page, query.PageSize, total)
}

// Show godoc
//
//	@Summary		Get a voucher
//	@Tags			vouchers
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Voucher ID"
//	@Success		200	{object}	response.Body{data=dto.VoucherResponse}
//	@Failure		404	{object}	response.Body
//	@Router			/admin/vouchers/{id} [get]
func (h *VoucherHandler) Show(c *gin.Context) {
	v, err := h.voucherService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

// Create godoc
//
//	@Summary		Create a voucher (pending_approval)
//	@Tags			vouchers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.VoucherRequest	true	"Voucher"
//	@Success		201		{object}	response.Body{data=dto.VoucherResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/vouchers [post]
func (h *VoucherHandler) Create(c *gin.Context) {
	var req dto.VoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	v, err := h.voucherService.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, v)
}

// Update godoc
//
//	@Summary		Update a voucher (only while pending_approval)
//	@Tags			vouchers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Voucher ID"
//	@Param			payload	body		dto.UpdateVoucherRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.VoucherResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/vouchers/{id} [put]
func (h *VoucherHandler) Update(c *gin.Context) {
	var req dto.UpdateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	v, err := h.voucherService.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

// SetStatus godoc
//
//	@Summary		Approve, reject or disable a voucher
//	@Tags			vouchers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Voucher ID"
//	@Param			payload	body		dto.VoucherStatusRequest	true	"New status"
//	@Success		200		{object}	response.Body{data=dto.VoucherResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/vouchers/{id}/status [put]
func (h *VoucherHandler) SetStatus(c *gin.Context) {
	var req dto.VoucherStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	v, err := h.voucherService.SetStatus(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req.Status)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, v)
}

// Conflicts godoc
//
//	@Summary		List redemptions where the redeemer is also the voucher's creator
//	@Tags			vouchers
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.VoucherConflictResponse}
//	@Router			/admin/vouchers/conflicts [get]
func (h *VoucherHandler) Conflicts(c *gin.Context) {
	rows, err := h.voucherService.Conflicts(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}
