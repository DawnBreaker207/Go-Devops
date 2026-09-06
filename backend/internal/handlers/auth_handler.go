package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// AuthHandler nhan request xac thuc, khong chua logic nghiep vu.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler tao AuthHandler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
//
//	@Summary		Dang ky tai khoan
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.RegisterRequest	true	"Thong tin dang ky"
//	@Success		201		{object}	response.Body{data=dto.UserResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, user)
}

// Login godoc
//
//	@Summary		Dang nhap
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.LoginRequest	true	"Email va mat khau"
//	@Success		200		{object}	response.Body{data=dto.LoginResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// Refresh godoc
//
//	@Summary		Lam moi access token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.RefreshRequest	true	"Refresh token"
//	@Success		200		{object}	response.Body{data=dto.TokenResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Router			/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}

	tokens, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, tokens)
}
