package models

import "time"

// RefreshToken is one issued refresh token (ID = JWT jti). Tokens rotated
// from the same login share a FamilyID, so a replay can revoke all of them.
type RefreshToken struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string     `gorm:"type:uuid;not null" json:"user_id"`
	FamilyID  string     `gorm:"type:uuid;not null" json:"family_id"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
