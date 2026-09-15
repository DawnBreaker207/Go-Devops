// Package audit writes audit rows in the same transaction as the business change.
//
// Action naming convention — <namespace>.<verb>, one action name shared by
// both the success row (written in-transaction by the service) and the
// failure row (written by middleware.Audit outside any transaction), so
// filtering by action always shows both outcomes together:
//
//   - auth.*     — identity: login, register, refresh, logout, password reset.
//   - users.*    — customer self-service on their own profile (change
//     password, update profile, delete_me).
//   - admin.*    — admin/staff mutating catalog & system config (movies,
//     halls, seats, showtimes, users, media, batch jobs).
//   - orders.*   — actions a customer or staff member initiates in a
//     booking's lifecycle (hold, pay, confirm, cancel, expire, counter_sell).
//   - payments.* — state the system/a webhook drives on its own (failed,
//     refunded, refund_stuck, abandoned, a rejected/forged callback).
//   - staff.*    — floor actions outside the standard order lifecycle
//     (ticket redeem, both the ok and the refused scan).
//
// Every event that belongs to a booking's lifecycle — regardless of which of
// booking/payment/ticket it is primarily about — should set Record.BookingID,
// so "the full history of order X" is one indexed query instead of chasing
// resource_id across three different resource_types.
package audit

import (
	"context"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

// Must match the ck_audit_outcome check constraint.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

type Record struct {
	ActorID      string // empty for webhook/system events
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	BookingID    string // set whenever this event belongs to a booking's lifecycle
	Before       map[string]any
	After        map[string]any
	IP           string
	UserAgent    string
	Outcome      string // defaults to OutcomeSuccess
	ErrorMessage string
}

type ctxKey struct{}

// ErrorMsgKey is the gin key where response.Error stores the message to audit.
const ErrorMsgKey = "audit_error_message"

// FromContext returns the Record stashed by middleware.Audit, if any.
func FromContext(ctx context.Context) (Record, bool) {
	r, ok := ctx.Value(ctxKey{}).(Record)
	return r, ok
}

func Stash(ctx context.Context, r Record) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

func ErrorMessage(c *gin.Context) string {
	msg, _ := c.Get(ErrorMsgKey)
	s, _ := msg.(string)
	return s
}

// In writes an audit row, usually in the open transaction. Callers must return its
// error so the business change rolls back with it.
func In(ctx context.Context, db *gorm.DB, r Record) error {
	if db == nil {
		return nil
	}
	outcome := r.Outcome
	if outcome == "" {
		outcome = OutcomeSuccess
	}
	// Cut to column sizes so an over-long value never fails the insert and rolls back the change.
	entry := &models.AuditLog{
		ActorID:      nullableUUID(r.ActorID),
		ActorRole:    truncate(r.ActorRole, 32),
		Action:       truncate(r.Action, 64),
		ResourceType: truncate(r.ResourceType, 64),
		ResourceID:   truncate(r.ResourceID, 128),
		BookingID:    nullableUUID(r.BookingID),
		BeforeJSON:   r.Before,
		AfterJSON:    r.After,
		IP:           truncate(r.IP, 64),
		UserAgent:    truncate(r.UserAgent, 512),
		Outcome:      outcome,
		ErrorMessage: truncate(r.ErrorMessage, 2000),
	}
	return db.WithContext(ctx).Create(entry).Error
}

// FromGin fills IP and User-Agent from the request only when they are empty.
func FromGin(c *gin.Context, r Record) Record {
	if r.IP == "" {
		r.IP = c.ClientIP()
	}
	if r.UserAgent == "" {
		r.UserAgent = c.GetHeader("User-Agent")
	}
	return r
}

// truncate counts runes because Postgres VARCHAR limits characters, not bytes.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func nullableUUID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
