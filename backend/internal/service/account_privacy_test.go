package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

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

	if err := e.accounts.DeleteMe(e.ctx, uid); !errors.Is(err, apperrors.ErrAccountHoldsTickets) {
		t.Fatalf("delete with tickets to come = %v", err)
	}

	e.moveShowStart(e.showID, -120*time.Minute)
	e.must(e.accounts.DeleteMe(e.ctx, uid))

	var anon models.User
	e.must(e.db.First(&anon, "id = ?", uid).Error)
	if anon.Email != "deleted-"+uid+"@anon.invalid" || anon.FullName != "" || anon.Phone != "" ||
		anon.Active || anon.Password == u.Password {
		t.Fatalf("anonymized user = %+v", anon)
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
}
