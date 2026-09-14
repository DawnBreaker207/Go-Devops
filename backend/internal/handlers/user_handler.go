package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Me godoc
//
//	@Summary		Current logged-in account info
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=dto.UserResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/users/me [get]
func (h *UserHandler) Me(c *gin.Context) {
	user, err := h.userService.GetByID(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, user)
}

// List godoc
//
//	@Summary		List accounts (admin)
//	@Tags			admin-users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int		false	"Page"
//	@Param			page_size	query	int		false	"Page size"
//	@Param			search		query	string	false	"Email or name"
//	@Param			role		query	string	false	"customer | staff | admin"
//	@Param			active		query	bool	false	"Filter by lock state"
//	@Success		200	{object}	response.Body{data=response.Paged{items=[]dto.UserResponse}}
//	@Failure		403	{object}	response.Body
//	@Router			/admin/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var query dto.UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	users, total, err := h.userService.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, users, query.Page, query.PageSize, total)
}

// Create godoc
//
//	@Summary		Create a staff or admin account (admin)
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.CreateUserRequest	true	"Account"
//	@Success		201		{object}	response.Body{data=dto.UserResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, err := h.userService.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, user)
}

// Update godoc
//
//	@Summary		Lock/unlock an account and/or change its role (admin)
//	@Description	Send active and/or role. A locked account can not log in, refresh, hold seats or pay; its confirmed tickets still pass the gate. Admins can not lock themselves, change their own role, or lock/demote the last active admin. Also served as PUT.
//	@Tags			admin-users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"User ID"
//	@Param			payload	body		dto.UpdateUserRequest	true	"Lock state"
//	@Success		200		{object}	response.Body{data=dto.UserResponse}
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/users/{id} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, err := h.userService.Update(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, user)
}
