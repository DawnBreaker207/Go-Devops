package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type BranchHandler struct {
	branches service.BranchService
}

func NewBranchHandler(branches service.BranchService) *BranchHandler {
	return &BranchHandler{branches: branches}
}

// List godoc
//
//	@Summary		List active branches (public — for the branch picker)
//	@Tags			branches
//	@Produce		json
//	@Success		200	{object}	response.Body{data=[]dto.BranchResponse}
//	@Router			/branches [get]
func (h *BranchHandler) List(c *gin.Context) {
	rows, err := h.branches.List(c.Request.Context(), true)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// ListAdmin godoc
//
//	@Summary		List all branches, including inactive (admin)
//	@Tags			branches
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.BranchResponse}
//	@Router			/admin/branches [get]
func (h *BranchHandler) ListAdmin(c *gin.Context) {
	rows, err := h.branches.List(c.Request.Context(), false)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// Create godoc
//
//	@Summary		Create a branch
//	@Tags			branches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.BranchRequest	true	"Branch"
//	@Success		201		{object}	response.Body{data=dto.BranchResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/branches [post]
func (h *BranchHandler) Create(c *gin.Context) {
	var req dto.BranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	b, err := h.branches.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, b)
}

// Update godoc
//
//	@Summary		Update a branch
//	@Tags			branches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Branch ID"
//	@Param			payload	body		dto.UpdateBranchRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.BranchResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/branches/{id} [put]
func (h *BranchHandler) Update(c *gin.Context) {
	var req dto.UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	b, err := h.branches.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, b)
}
