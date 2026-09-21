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
	req.UserAgent = c.Request.UserAgent()

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

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
	req.UserAgent = c.Request.UserAgent()

	result, err := h.authService.AcceptTerms(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, result)
}

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

//	@Summary		List signed-in devices
//	@Description	Pass device_id (the caller's own, if known) to flag it as is_current.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			device_id	query	string	false	"Caller's own device id"
//	@Success		200	{object}	response.Body{data=[]dto.SessionResponse}
//	@Failure		401	{object}	response.Body
//	@Router			/users/me/sessions [get]
func (h *AuthHandler) ListSessions(c *gin.Context) {
	var query dto.SessionListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	sessions, err := h.authService.ListSessions(c.Request.Context(), middleware.CurrentUserID(c), query.DeviceID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, sessions)
}

//	@Summary		Sign out one device
//	@Description	Revokes exactly one session; the account's other devices stay signed in.
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Session (refresh token) id"
//	@Success		200	{object}	response.Body
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body	"session does not belong to the caller"
//	@Router			/users/me/sessions/{id} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	if err := h.authService.RevokeSession(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil)
}
