package dto

import (
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// AuditLogListQuery filters the admin audit trail. from/to bound created_at
// (RFC 3339 or YYYY-MM-DD), the same convention as the daily report.
type AuditLogListQuery struct {
	PageQuery
	Action       string `form:"action" binding:"omitempty,max=64"`
	ResourceType string `form:"resource_type" binding:"omitempty,max=64"`
	ResourceID   string `form:"resource_id" binding:"omitempty,max=128"`
	BookingID    string `form:"booking_id" binding:"omitempty,uuid"`
	ActorID      string `form:"actor_id" binding:"omitempty,uuid"`
	Outcome      string `form:"outcome" binding:"omitempty,oneof=success failure"`
	From         string `form:"from" binding:"omitempty"`
	To           string `form:"to" binding:"omitempty"`
}

// AuditLogResponse mirrors models.AuditLog for the admin read API.
type AuditLogResponse struct {
	ID           string         `json:"id"`
	ActorID      string         `json:"actor_id,omitempty"`
	ActorRole    string         `json:"actor_role,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id,omitempty"`
	BookingID    string         `json:"booking_id,omitempty"`
	BeforeJSON   map[string]any `json:"before_json,omitempty"`
	AfterJSON    map[string]any `json:"after_json,omitempty"`
	IP           string         `json:"ip,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	Outcome      string         `json:"outcome"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

func NewAuditLogResponse(a models.AuditLog) AuditLogResponse {
	resp := AuditLogResponse{
		ID:           a.ID,
		ActorRole:    a.ActorRole,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		BeforeJSON:   a.BeforeJSON,
		AfterJSON:    a.AfterJSON,
		IP:           a.IP,
		UserAgent:    a.UserAgent,
		Outcome:      a.Outcome,
		ErrorMessage: a.ErrorMessage,
		CreatedAt:    a.CreatedAt,
	}
	if a.ActorID != nil {
		resp.ActorID = *a.ActorID
	}
	if a.BookingID != nil {
		resp.BookingID = *a.BookingID
	}
	return resp
}
