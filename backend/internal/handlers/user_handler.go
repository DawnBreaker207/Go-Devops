package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// UserHandler nhan request lien quan den nguoi dung.
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler tao UserHandler.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Me godoc
//
//	@Summary		Thong tin tai khoan dang dang nhap
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=github_com_Cinema-Project-Juann_BackEnd-CP_internal_dto.UserResponse}
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
