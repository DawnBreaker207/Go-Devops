package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// UserResponse la thong tin nguoi dung tra ve cho client (khong co password).
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUserResponse map model sang DTO.
func NewUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
