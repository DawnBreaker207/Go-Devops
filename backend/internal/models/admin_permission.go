package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Four separate admin permission groups (Phần 9.2 of ADVANCED_FEATURES_DISCUSSION.md):
// no admin has all four by default, and only the owner role may grant/revoke them.
const (
	PermissionContent  = "content"
	PermissionPricing  = "pricing"
	PermissionFinance  = "finance"
	PermissionAccounts = "accounts"
)

var AllPermissions = []string{PermissionContent, PermissionPricing, PermissionFinance, PermissionAccounts}

type AdminPermission struct {
	ID            string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        string    `gorm:"type:uuid;not null" json:"user_id"`
	PermissionKey string    `gorm:"type:varchar(16);not null" json:"permission_key"`
	GrantedBy     string    `gorm:"type:uuid;not null" json:"granted_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func (AdminPermission) TableName() string { return "admin_permissions" }

func (p *AdminPermission) BeforeCreate(*gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
