package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role is the user's access level.
const (
	RoleAdmin    = "admin"
	RoleStaff    = "staff"
	RoleCustomer = "customer"
)

// User is a system user.
type User struct {
	ID        string         `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`
	FullName  string         `gorm:"type:varchar(255);not null" json:"full_name"`
	Role      string         `gorm:"type:varchar(32);not null;default:customer" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName fixes the table name to bypass GORM's naming convention.
func (User) TableName() string { return "users" }

// BeforeCreate generates a UUID if none is set.
func (u *User) BeforeCreate(*gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	return nil
}
