package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type OwnerHandler struct {
	ownerService service.OwnerService
}

func NewOwnerHandler(ownerService service.OwnerService) *OwnerHandler {
	return &OwnerHandler{ownerService: ownerService}
}

// SetOwner godoc
//
//	@Summary		Promote an admin to owner, or demote an owner back to admin
//	@Description	Owner-only. At least one active owner must always remain.
//	@Tags			owner
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string				true	"User ID"
//	@Param			payload	body	dto.SetOwnerRequest	true	"owner: true to promote, false to demote"
//	@Success		200		{object}	response.Body{data=dto.UserResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/owners/{id} [patch]
func (h *OwnerHandler) SetOwner(c *gin.Context) {
	var req dto.SetOwnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, err := h.ownerService.SetOwner(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req.Owner)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}

// GrantPermission godoc
//
//	@Summary		Grant an admin permission group
//	@Description	Owner-only. Deny-by-default: a fresh admin has none of the 4 groups until granted here.
//	@Tags			owner
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path	string					true	"Admin user ID"
//	@Param			payload	body	dto.PermissionRequest	true	"Permission key"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/owners/{id}/permissions [post]
func (h *OwnerHandler) GrantPermission(c *gin.Context) {
	var req dto.PermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.ownerService.Grant(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req.PermissionKey); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "granted")
}

// RevokePermission godoc
//
//	@Summary		Revoke an admin permission group
//	@Tags			owner
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Admin user ID"
//	@Param			key	path	string	true	"Permission key"
//	@Success		200	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Router			/admin/owners/{id}/permissions/{key} [delete]
func (h *OwnerHandler) RevokePermission(c *gin.Context) {
	if err := h.ownerService.Revoke(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), c.Param("key")); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "revoked")
}

// ListPermissions godoc
//
//	@Summary		List an admin's permission groups
//	@Tags			owner
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Admin user ID"
//	@Success		200	{object}	response.Body{data=[]dto.AdminPermissionResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Router			/admin/owners/{id}/permissions [get]
func (h *OwnerHandler) ListPermissions(c *gin.Context) {
	perms, err := h.ownerService.ListPermissions(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, perms)
}
