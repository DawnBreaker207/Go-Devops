package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func NewUserResponses(users []models.User) []UserResponse {
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, NewUserResponse(&users[i]))
	}
	return out
}

type UserListQuery struct {
	PageQuery
	Role   string `form:"role" binding:"omitempty,oneof=customer staff admin"`
	Active *bool  `form:"active"`
}

// Staff (or admin) only; customers self-register.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email,max=255" example:"staff1@cinema.local"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	FullName string `json:"full_name" binding:"required,min=2,max=255"`
	Role     string `json:"role" binding:"required,oneof=staff admin" example:"staff"`
}

// Send at least one field.
type UpdateUserRequest struct {
	Active *bool   `json:"active"`
	Role   *string `json:"role" binding:"omitempty,oneof=customer staff admin" example:"staff"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=255" example:"Nguyen Van A"`
	Phone    string `json:"phone" binding:"omitempty,max=20" example:"0901234567"`
}

// Re-confirms the password before irreversible erasure.
type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required" example:"secret123"`
}

// No row yet reads as model defaults.
type NotificationPreferenceResponse struct {
	BookingReminders bool `json:"booking_reminders"`
	PromoOffers      bool `json:"promo_offers"`
}

func NewNotificationPreferenceResponse(p *models.NotificationPreference) NotificationPreferenceResponse {
	return NotificationPreferenceResponse{
		BookingReminders: p.BookingReminders,
		PromoOffers:      p.PromoOffers,
	}
}

// Replaces both flags at once; both false is allowed.
type UpdateNotificationPreferenceRequest struct {
	BookingReminders bool `json:"booking_reminders"`
	PromoOffers      bool `json:"promo_offers"`
}
