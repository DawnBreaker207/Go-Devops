package dto

import "time"

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"admin@cinema.local"`
	Password string `json:"password" binding:"required,min=6,max=72" example:"secret123"`
	FullName string `json:"full_name" binding:"required,min=2,max=255" example:"Nguyen Van A"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"admin@cinema.local"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
	// DeviceID is generated and kept by the frontend so the resulting session can
	// later be listed and individually signed out from /users/me/sessions.
	DeviceID string `json:"device_id" binding:"omitempty,max=255" example:"web-3f0c9b6e"`
	// ClientIP keys the failed-login lockout together with the email.
	ClientIP string `json:"-"`
	// UserAgent is read from the request header by the handler, not the body.
	UserAgent string `json:"-"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=255" example:"admin@cinema.local"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=72" example:"newsecret123"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=72" example:"newsecret123"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

type LoginResponse struct {
	TokenResponse
	User UserResponse `json:"user"`
}

// SessionResponse is one signed-in device, as listed by GET /users/me/sessions.
type SessionResponse struct {
	ID         string     `json:"id"`
	UserAgent  string     `json:"user_agent,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	// IsCurrent is true only when the caller told us its device_id and it matches.
	IsCurrent bool `json:"is_current"`
}

// SessionListQuery optionally identifies the caller's own device so the response
// can flag is_current.
type SessionListQuery struct {
	DeviceID string `form:"device_id" binding:"omitempty,max=255"`
}
