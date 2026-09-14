// Package audit records an activity trail. Rows are written inside the same
// transaction as the business change, so a log row never exists without its
// data. Logs are read straight from the DB (no public endpoint yet).
package audit

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Supported outcomes (kept in sync with ck_audit_outcome).
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

// Record is one event to persist.
type Record struct {
	ActorID      string // empty for webhook/system events
	ActorRole    string
	Action       string // e.g. "hold_seats", "payment_confirmed", "admin.update_hall"
	ResourceType string // e.g. "booking", "hall", "showtime", "user"
	ResourceID   string
	Before       map[string]any
	After        map[string]any
	IP           string
	UserAgent    string
	Outcome      string // OutcomeSuccess on writes in-transaction; set by failure writer
	ErrorMessage string
}

// ctxKey guards the per-request Record carried across middleware.
type ctxKey struct{}

// ErrorMsgKey is the gin Keys slot where response.Error stashes the message
// a failing request should be audited with.
const ErrorMsgKey = "audit_error_message"

// FromContext returns the Record stashed by middleware.Audit, if any.
func FromContext(ctx context.Context) (Record, bool) {
	r, ok := ctx.Value(ctxKey{}).(Record)
	return r, ok
}

// Stash returns a context carrying the Record for the request, so services
// can pick it up with FromContext and write the success row.
func Stash(ctx context.Context, r Record) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

// ErrorMessage returns the error message stashed on a failed response.
func ErrorMessage(c *gin.Context) string {
	msg, _ := c.Get(ErrorMsgKey)
	s, _ := msg.(string)
	return s
}

// In writes an audit row to db (usually the open transaction). Errors bubble
// up to roll back with the business change — never swallowed.
func In(ctx context.Context, db *gorm.DB, r Record) error {
	if db == nil {
		return nil
	}
	outcome := r.Outcome
	if outcome == "" {
		outcome = OutcomeSuccess
	}
	entry := &models.AuditLog{
		ActorID:      nullableUUID(r.ActorID),
		ActorRole:    r.ActorRole,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		BeforeJSON:   r.Before,
		AfterJSON:    r.After,
		IP:           r.IP,
		UserAgent:    r.UserAgent,
		Outcome:      outcome,
		ErrorMessage: r.ErrorMessage,
	}
	return db.WithContext(ctx).Create(entry).Error
}

// FromGin fills IP and User-Agent from the current request if missing.
func FromGin(c *gin.Context, r Record) Record {
	if r.IP == "" {
		r.IP = c.ClientIP()
	}
	if r.UserAgent == "" {
		r.UserAgent = c.GetHeader("User-Agent")
	}
	return r
}

func nullableUUID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}