package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// UserResponse is user info returned to the client (no password).
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	BranchID  string    `json:"branch_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user *models.User) UserResponse {
	res := UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	if user.BranchID != nil {
		res.BranchID = *user.BranchID
	}
	return res
}

func NewUserResponses(users []models.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, NewUserResponse(&users[i]))
	}
	return out
}

// UserListQuery filters the admin user list.
type UserListQuery struct {
	PageQuery
	Role   string `form:"role" binding:"omitempty,oneof=customer staff admin"`
	Active *bool  `form:"active"`
}

// CreateUserRequest creates a staff (or admin) account; customers register
// themselves
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"staff1@cinema.local"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	FullName string `json:"full_name" binding:"required,min=2,max=255"`
	Role     string `json:"role" binding:"required,oneof=staff admin" example:"staff"`
}

// UpdateUserRequest locks/unlocks an account and/or changes its role; send at
// least one field.
type UpdateUserRequest struct {
	Active *bool   `json:"active"`
	Role   *string `json:"role" binding:"omitempty,oneof=customer staff admin" example:"staff"`
	// BranchID scopes a staff account to one branch (Phần 4); only settable,
	// not clearable, through this endpoint (send a real id to assign one).
	BranchID *string `json:"branch_id" binding:"omitempty"`
}

// UpdateProfileRequest is what a signed-in user may change about themselves.
type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=255" example:"Nguyen Van A"`
	Phone    string `json:"phone" binding:"omitempty,max=20" example:"0901234567"`
}

// DeleteAccountRequest re-confirms the password before an irreversible erasure.
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required" example:"secret123"`
}
