package dto

import "time"

// SetOwnerRequest promotes an existing admin to owner (Owner: true) or demotes
// an owner back to admin (Owner: false). Only role=owner may call this.
type SetOwnerRequest struct {
	Owner bool `json:"owner"`
}

// PermissionRequest grants or revokes one of the four permission groups
// (content/pricing/finance/accounts). Only role=owner may call this.
type PermissionRequest struct {
	PermissionKey string `json:"permission_key" binding:"required,oneof=content pricing finance accounts" example:"pricing"`
}

type AdminPermissionResponse struct {
	UserID        string    `json:"user_id"`
	PermissionKey string    `json:"permission_key"`
	GrantedBy     string    `json:"granted_by"`
	CreatedAt     time.Time `json:"created_at"`
}
