package service_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

var resetTokenPattern = regexp.MustCompile(`token=([0-9a-f]{64})`)

// T60: logout revokes the whole refresh family, not just the presented token.
func TestLogout_RevokesWholeFamily(t *testing.T) {
	e := newEnv(t)
	e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	res, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)
	first := res.RefreshToken
	second, err := e.auth.Refresh(e.ctx, first)
	e.must(err)

	if err := e.auth.Logout(e.ctx, second.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := e.auth.Refresh(e.ctx, first); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("family replay after logout: err = %v, want 401", err)
	}
	if err := e.auth.Logout(e.ctx, first); err != nil {
		t.Fatalf("logout an already-dead token: %v", err)
	}

	again, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)
	if _, err := e.auth.Refresh(e.ctx, again.RefreshToken); err != nil {
		t.Fatalf("new login family: %v", err)
	}
}

// T61: changing the password refuses the wrong current one, then revokes every
// old session and hands the current one a fresh pair working with the new secret.
func TestChangePassword_RotatesPasswordAndSessions(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "c@test.local", "secret123")
	before, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)

	if _, err := e.auth.ChangePassword(e.ctx, u.ID,
		dto.ChangePasswordRequest{CurrentPassword: "wrong", NewPassword: "nextpass123"}); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("wrong current: err = %v, want 401", err)
	}

	after, err := e.auth.ChangePassword(e.ctx, u.ID,
		dto.ChangePasswordRequest{CurrentPassword: "secret123", NewPassword: "nextpass123"})
	e.must(err)
	if after.RefreshToken == "" || after.RefreshToken == before.RefreshToken {
		t.Fatalf("tokens did not rotate: before=%q after=%q", before.RefreshToken, after.RefreshToken)
	}

	if _, err := e.auth.Refresh(e.ctx, before.RefreshToken); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("old session after change: err = %v, want 401", err)
	}
	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1")); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("old password: err = %v, want 401", err)
	}
	if _, err := e.auth.Refresh(e.ctx, after.RefreshToken); err != nil {
		t.Fatalf("current session rotates after change: %v", err)
	}
	relogin, err := e.auth.Login(e.ctx, login("c@test.local", "nextpass123", "10.0.0.1"))
	e.must(err)
	if _, err := e.auth.Refresh(e.ctx, relogin.RefreshToken); err != nil {
		t.Fatalf("new password login session: %v", err)
	}
}

// T62: password reset is single-use; a live token redeems once, the old password
// dies, and every session is revoked while timing blows up for missing accounts.
func TestForgotPassword_ResetIsSingleUseAndRevokesSessions(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "c@test.local", "secret123")
	session, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)

	if err := e.auth.ForgotPassword(e.ctx, "c@Test.local"); err != nil {
		t.Fatal(err)
	}
	msgs := e.mailer.Sent()
	if len(msgs) != 1 || msgs[0].To != "c@test.local" || !strings.HasPrefix(msgs[0].Subject, "Đặt lại mật khẩu") {
		t.Fatalf("reset email = %+v", msgs)
	}
	m := resetTokenPattern.FindStringSubmatch(msgs[0].HTML)
	if m == nil {
		t.Fatal("no reset token in the email")
	}
	raw := m[1]

	if n := e.count(`SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = ?`, u.ID); n != 1 {
		t.Fatalf("reset tokens = %d, want 1", n)
	}

	if err := e.auth.ResetPassword(e.ctx, dto.ResetPasswordRequest{Token: strings.Repeat("0", 64), NewPassword: "freshpass123"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("bogus token: err = %v, want 400", err)
	}
	if err := e.auth.ResetPassword(e.ctx, dto.ResetPasswordRequest{Token: strings.Repeat("0", 40), NewPassword: "freshpass123"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("short token: err = %v, want 400", err)
	}

	if err := e.auth.ResetPassword(e.ctx, dto.ResetPasswordRequest{Token: raw, NewPassword: "freshpass123"}); err != nil {
		t.Fatal(err)
	}
	if err := e.auth.ResetPassword(e.ctx, dto.ResetPasswordRequest{Token: raw, NewPassword: "freshpass123"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("token reuse: err = %v, want 400", err)
	}
	if _, err := e.auth.Refresh(e.ctx, session.RefreshToken); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("session after reset: err = %v, want 401", err)
	}
	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1")); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("old password after reset: err = %v, want 401", err)
	}
	relogin, err := e.auth.Login(e.ctx, login("c@test.local", "freshpass123", "10.0.0.1"))
	e.must(err)
	if _, err := e.auth.Refresh(e.ctx, relogin.RefreshToken); err != nil {
		t.Fatalf("session after reset: %v", err)
	}
}

// T62b: an expired token is refused even though it has never been used.
func TestForgotPassword_ExpiredTokenRefused(t *testing.T) {
	e := newEnv(t)
	e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	if err := e.auth.ForgotPassword(e.ctx, "c@test.local"); err != nil {
		t.Fatal(err)
	}
	m := resetTokenPattern.FindStringSubmatch(e.mailer.Sent()[0].HTML)
	if m == nil {
		t.Fatal("no reset token in the email")
	}
	raw := m[1]
	e.must(e.db.Exec(`UPDATE password_reset_tokens SET expires_at = NOW() - INTERVAL '1 minute'`).Error)

	if err := e.auth.ResetPassword(e.ctx, dto.ResetPasswordRequest{Token: raw, NewPassword: "freshpass123"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("expired token: err = %v, want 400", err)
	}
}

// T62c: unknown and locked accounts answer 200 and never trigger an email.
func TestForgotPassword_Privacy(t *testing.T) {
	e := newEnv(t)
	locked := e.newUser(models.RoleCustomer, "locked@test.local", "secret123")
	e.must(e.db.Model(&models.User{}).Where("id = ?", locked.ID).Update("active", false).Error)

	if err := e.auth.ForgotPassword(e.ctx, "nobody@test.local"); err != nil {
		t.Fatal("unknown account must answer 200")
	}
	if err := e.auth.ForgotPassword(e.ctx, "locked@test.local"); err != nil {
		t.Fatal("locked account must answer 200")
	}
	if n := len(e.mailer.Sent()); n != 0 {
		t.Fatalf("emails sent for unknown/locked accounts: %d", n)
	}
}

// T63: the caller may fix their own name and phone; a phone is validated and can be cleared.
func TestUpdateProfile_ValidatesAndPersists(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	if _, err := e.accounts.UpdateProfile(e.ctx, u.ID,
		dto.UpdateProfileRequest{FullName: "New Name", Phone: "+84short"}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("bad phone: err = %v, want 400", err)
	}
	got, err := e.accounts.UpdateProfile(e.ctx, u.ID,
		dto.UpdateProfileRequest{FullName: "  New Name  ", Phone: "+84901234567"})
	e.must(err)
	if got.FullName != "New Name" || got.Phone != "+84901234567" {
		t.Fatalf("profile = %+v", got)
	}
	var phone string
	e.must(e.db.Raw(`SELECT COALESCE(phone, '') FROM users WHERE id = ?`, u.ID).Scan(&phone).Error)
	if phone != "+84901234567" {
		t.Fatalf("db phone = %q", phone)
	}

	cleared, err := e.accounts.UpdateProfile(e.ctx, u.ID, dto.UpdateProfileRequest{FullName: "New Name"})
	e.must(err)
	if cleared.Phone != "" {
		t.Fatalf("cleared phone = %q", cleared.Phone)
	}
	e.must(e.db.Raw(`SELECT COALESCE(phone, '') FROM users WHERE id = ?`, u.ID).Scan(&phone).Error)
	if phone != "" {
		t.Fatalf("db phone still %q after clear", phone)
	}
}
