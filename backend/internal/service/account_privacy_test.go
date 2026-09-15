package service_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// An erased account is gone, not just locked: an admin can no longer "find"
// it to flip active back on, unlike a plain lock (Update active=false).
func TestAccount_DeleteMeBlocksReactivation(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "erased-reactivate@test.local", "secret123")
	admin := e.newUser(models.RoleAdmin, "admin-reactivate@test.local", "secret123")

	e.must(e.accounts.DeleteMe(e.ctx, u.ID, "secret123"))

	if _, err := e.accounts.Update(e.ctx, admin.ID, u.ID, setActive(true)); !isAppErr(err, apperrors.ErrUserNotFound) {
		t.Fatalf("admin reactivating an erased account = %v, want ErrUserNotFound", err)
	}
	if _, err := e.accounts.GetByID(e.ctx, u.ID); !isAppErr(err, apperrors.ErrUserNotFound) {
		t.Fatalf("GetByID an erased account = %v, want ErrUserNotFound", err)
	}
	list, _, err := e.accounts.List(e.ctx, dto.UserListQuery{PageQuery: dto.PageQuery{Page: 1, PageSize: 50}})
	e.must(err)
	for _, row := range list {
		if row.ID == u.ID {
			t.Fatalf("erased account %s still appears in the admin list", u.ID)
		}
	}
}

// DeleteMe must invalidate the account status cache immediately, the same
// way Update (lock/unlock) already does — otherwise a still-valid access
// token for the just-erased account keeps passing the cached check for up
// to the cache TTL.
func TestAccount_DeleteMeInvalidatesStatusCacheAtOnce(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "erased-cache@test.local", "secret123")

	cache := service.NewAccountStatusCache(repository.NewUserRepository(e.db), time.Hour)
	accounts := service.NewUserService(e.db, repository.NewUserRepository(e.db), cache.Invalidate)

	// Warm the cache with the pre-erasure (active) status.
	active, _, err := cache.Status(e.ctx, u.ID)
	e.must(err)
	if !active {
		t.Fatal("control: account should start active")
	}

	e.must(accounts.DeleteMe(e.ctx, u.ID, "secret123"))

	if _, _, err := cache.Status(e.ctx, u.ID); !errors.Is(err, apperrors.ErrInvalidToken) {
		t.Fatalf("status right after erasure = %v, want ErrInvalidToken (not a stale cached active=true)", err)
	}
}

// T65: with terms enforcement on, login answers 428 until the account accepts
// the current revision; terms-accept re-verifies the password and signs in.
func TestAuth_TermsGateForcesAcceptance(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "terms@test.local", "secret123")
	req := dto.LoginRequest{Email: u.Email, Password: "secret123", ClientIP: "10.0.0.1"}

	res, err := e.auth.Login(e.ctx, req)
	e.must(err)
	if res.User.ID != u.ID {
		t.Fatalf("control login:\n\tgot  %s\n\twant %s", res.User.ID, u.ID)
	}

	termsAuth := service.NewAuthService(e.db, repository.NewUserRepository(e.db),
		repository.NewRefreshTokenRepository(e.db), e.jwt, nil,
		repository.NewPasswordResetTokenRepository(e.db), e.mailer,
		"http://test.local/reset-password", 30*time.Minute, 1)

	_, err = termsAuth.Login(e.ctx, req)
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperrors.CodeTermsRequired {
		t.Fatalf("login without terms = %v", err)
	}
	if appErr.Details["required_terms_version"] != "1" || appErr.Details["accepted_terms_version"] != "0" {
		t.Fatalf("terms details = %v", appErr.Details)
	}

	_, err = termsAuth.AcceptTerms(e.ctx, dto.LoginRequest{Email: u.Email, Password: "nope", ClientIP: "10.0.0.1"})
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("terms-accept with wrong password = %v", err)
	}

	accept, err := termsAuth.AcceptTerms(e.ctx, req)
	e.must(err)
	if accept.User.ID != u.ID {
		t.Fatalf("terms-accept user = %s", accept.User.ID)
	}
	var stored models.User
	e.must(e.db.First(&stored, "id = ?", u.ID).Error)
	if stored.AcceptedTermsVersion != 1 {
		t.Fatalf("accepted_terms_version = %d", stored.AcceptedTermsVersion)
	}
	if _, err := termsAuth.AcceptTerms(e.ctx, req); err == nil {
		t.Fatal("second terms-accept must fail")
	}
	if _, err := termsAuth.Login(e.ctx, req); err != nil {
		t.Fatalf("login after acceptance: %v", err)
	}
}

// T66: the right to erasure is refused while a confirmed ticket is still to
// come; afterwards the account is scrubbed and all sessions dropped.
func TestAccount_DeleteMeErasesAfterTicketsUsed(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "eraseme@test.local", "secret123")
	uid := u.ID

	confirmed := e.confirmed(uid, "A1")
	var b models.Booking
	e.must(e.db.First(&b, "id = ?", confirmed).Error)
	if b.Status != models.BookingConfirmed {
		t.Fatalf("status = %s", b.Status)
	}

	held, err := e.svc.Hold(e.ctx, uid, dto.HoldRequest{
		ShowID: e.showID, SeatIDs: e.ids("A2"), IdempotencyKey: "delete-pending"})
	e.must(err)

	e.auth.Login(e.ctx, dto.LoginRequest{Email: u.Email, Password: "secret123", ClientIP: "10.0.0.1"})
	e.must(e.auth.ForgotPassword(e.ctx, u.Email))

	if err := e.accounts.DeleteMe(e.ctx, uid, "secret123"); !errors.Is(err, apperrors.ErrAccountHoldsTickets) {
		t.Fatalf("delete with tickets to come = %v", err)
	}
	if err := e.accounts.DeleteMe(e.ctx, uid, "wrong-password"); !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("delete with the wrong password = %v", err)
	}

	e.moveShowStart(e.showID, -120*time.Minute)
	// Stash an audit context like middleware.Audit would in production, so
	// the users.delete_me row is actually written (DeleteMe only writes it
	// when a Record is already stashed on the context).
	auditCtx := audit.Stash(e.ctx, audit.Record{ActorID: uid, ActorRole: "customer", Action: "users.delete_me", ResourceType: "user"})
	e.must(e.accounts.DeleteMe(auditCtx, uid, "secret123"))

	// Unscoped: the row is now soft-deleted, so a plain query no longer finds it.
	var anon models.User
	e.must(e.db.Unscoped().First(&anon, "id = ?", uid).Error)
	if anon.Email != "deleted-"+uid+"@anon.invalid" || anon.FullName != "" || anon.Phone != "" ||
		anon.Active || anon.Password == u.Password || !anon.DeletedAt.Valid {
		t.Fatalf("anonymized user = %+v", anon)
	}

	// Erased, not just locked: a plain (scoped) lookup no longer finds the
	// account at all, so nothing — including List/GetByID/Update — can see it.
	var count int64
	e.must(e.db.Model(&models.User{}).Where("id = ?", uid).Count(&count).Error)
	if count != 0 {
		t.Fatalf("erased user still visible to a default query: count=%d", count)
	}

	var refreshCount, resetCount int64
	e.must(e.db.Model(&models.RefreshToken{}).Where("user_id = ?", uid).Count(&refreshCount).Error)
	e.must(e.db.Model(&models.PasswordResetToken{}).Where("user_id = ?", uid).Count(&resetCount).Error)
	if refreshCount != 0 || resetCount != 0 {
		t.Fatalf("sessions left: refresh=%d reset=%d", refreshCount, resetCount)
	}
	var pending models.Booking
	e.must(e.db.First(&pending, "id = ?", held.BookingID).Error)
	if pending.Status != models.BookingExpired {
		t.Fatalf("pending booking status = %s", pending.Status)
	}

	if _, err := e.auth.Login(e.ctx, dto.LoginRequest{Email: u.Email, Password: "secret123", ClientIP: "10.0.0.1"}); !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("login after erasure = %v", err)
	}

	// The audit trail of an erasure must not itself keep the erased email —
	// that would defeat the whole point of the right to erasure.
	var beforeJSON, afterJSON string
	e.must(e.db.Raw(`SELECT COALESCE(before_json::text, ''), COALESCE(after_json::text, '')
		FROM audit_logs WHERE action = 'users.delete_me' AND resource_id = ?`, uid).
		Row().Scan(&beforeJSON, &afterJSON))
	if strings.Contains(beforeJSON, u.Email) || strings.Contains(afterJSON, u.Email) {
		t.Fatalf("delete_me audit row still holds the real email: before=%s after=%s", beforeJSON, afterJSON)
	}
}
