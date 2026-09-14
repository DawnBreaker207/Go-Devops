package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Success rows are written inside the business transaction, so a log row never exists without
// its data; failure rows are written afterwards by the audit middleware.
type AuditLog struct {
	ID           string         `gorm:"type:uuid;primaryKey" json:"id"`
	ActorID      *string        `gorm:"type:uuid" json:"actor_id,omitempty"`
	ActorRole    string         `gorm:"type:varchar(32)" json:"actor_role,omitempty"`
	Action       string         `gorm:"type:varchar(64);not null" json:"action"`
	ResourceType string         `gorm:"type:varchar(64);not null" json:"resource_type"`
	ResourceID   string         `gorm:"type:varchar(128)" json:"resource_id,omitempty"`
	BeforeJSON   map[string]any `gorm:"serializer:json;type:jsonb" json:"before_json,omitempty"`
	AfterJSON    map[string]any `gorm:"serializer:json;type:jsonb" json:"after_json,omitempty"`
	IP           string         `gorm:"type:varchar(64)" json:"ip,omitempty"`
	UserAgent    string         `gorm:"type:varchar(512)" json:"user_agent,omitempty"`
	Outcome      string         `gorm:"type:varchar(16);not null;default:success" json:"outcome"`
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func (a *AuditLog) BeforeCreate(*gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}
