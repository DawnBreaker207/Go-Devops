package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
//
//	@Summary		Register account
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.RegisterRequest	true	"Registration details"
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
//	@Summary		Login
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.LoginRequest	true	"Email and password"
//	@Success		200		{object}	response.Body{data=dto.LoginResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body	"account locked"
//	@Failure		429		{object}	response.Body	"too many failed attempts"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ClientIP = c.ClientIP()

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// AcceptTerms godoc
//
//	@Summary		Accept the current terms and sign in
//	@Description	Verifies the credentials, records the accepted terms revision and issues a token pair. Needed when login answers 428.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body	dto.LoginRequest	true	"Email and password"
//	@Success		200		{object}	response.Body{data=dto.LoginResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		428		{object}	response.Body	"terms not required or already accepted"
//	@Router			/auth/terms-accept [post]
func (h *AuthHandler) AcceptTerms(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ClientIP = c.ClientIP()

	result, err := h.authService.AcceptTerms(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

// Refresh godoc
//
//	@Summary		Refresh access token
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

// Logout godoc
//
//	@Summary		Sign out: revokes the whole session family
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.LogoutRequest	true	"Refresh token to revoke"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body	"invalid or expired token"
//	@Router			/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ForgotPassword godoc
//
//	@Summary		Request a password reset email
//	@Description	Always answers 200 so the existence of an account is never leaked.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.ForgotPasswordRequest	true	"Account email"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Router			/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ResetPassword godoc
//
//	@Summary		Redeem a password reset token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		dto.ResetPasswordRequest	true	"Reset token and new password"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body	"invalid, expired, or already used token"
//	@Router			/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.authService.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}

// ChangePassword godoc
//
//	@Summary		Change the current password
//	@Description	Revokes all other sessions; the current one gets a fresh token pair.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.ChangePasswordRequest	true	"Current and new password"
//	@Success		200		{object}	response.Body{data=dto.TokenResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		401		{object}	response.Body	"wrong current password"
//	@Router			/users/me/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	tokens, err := h.authService.ChangePassword(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tokens)
}
