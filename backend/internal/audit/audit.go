// Package audit writes audit rows in the same transaction as the business change.
//
// Action naming — <namespace>.<verb>, shared by the in-transaction success row
// (service) and the failure row (middleware.Audit), so filtering shows both:
//
//   - auth.*     — identity: login, register, refresh, logout, password reset.
//   - users.*    — customer self-service (password, profile, delete_me).
//   - admin.*    — catalog & system config writes.
//   - orders.*   — booking lifecycle (hold, pay, confirm, cancel, expire, counter_sell).
//   - payments.* — system/webhook-driven state (failed, refunded, refund_stuck, abandoned, rejected callback).
//   - staff.*    — floor actions outside the lifecycle (ticket redeem, ok and refused).
//
// Lifecycle events should set Record.BookingID, so one query shows an order's full history.
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

// In writes an audit row; callers must return its error so the change rolls back with it.
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
