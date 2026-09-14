package dto

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"admin@cinema.local"`
	Password string `json:"password" binding:"required,min=6,max=72" example:"secret123"`
	FullName string `json:"full_name" binding:"required,min=2,max=255" example:"Nguyen Van A"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"admin@cinema.local"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
	// ClientIP keys the failed-login lockout together with the email.
	ClientIP string `json:"-"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
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
