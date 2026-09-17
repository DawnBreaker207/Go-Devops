package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	RoleOwner    = "owner"
	RoleAdmin    = "admin"
	RoleStaff    = "staff"
	RoleCustomer = "customer"
)

type User struct {
	ID       string `gorm:"type:uuid;primaryKey" json:"id"`
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password string `gorm:"type:varchar(255);not null" json:"-"`
	FullName string `gorm:"type:varchar(255);not null" json:"full_name"`
	Phone    string `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Role     string `gorm:"type:varchar(32);not null;default:customer" json:"role"`
	// AcceptedTermsVersion is the terms revision the account agreed to; a user
	// with AcceptedTermsVersion < the configured version can not log in.
	AcceptedTermsVersion int             `gorm:"not null;default:0" json:"accepted_terms_version"`
	// Active false locks the account: no login, refresh, hold or pay.
	Active bool `gorm:"not null;default:true" json:"active"`
	// BranchID is only meaningful for role=staff (Phần 4): nil means
	// unscoped (serves/sees every branch).
	BranchID  *string        `gorm:"type:uuid" json:"branch_id,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
