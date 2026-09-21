package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"html/template"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/notify"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
)

// dummyPasswordHash has the cost of real password hashes.
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.DefaultCost)
	return hash
})

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	AcceptTerms(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) (*dto.TokenResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error
	// ListSessions: caller's devices; currentDeviceID (may be empty) flags is_current.
	ListSessions(ctx context.Context, userID, currentDeviceID string) ([]dto.SessionResponse, error)
	// RevokeSession signs out one device; ErrSessionNotFound if not userID's.
	RevokeSession(ctx context.Context, userID, sessionID string) error
}

type authService struct {
	db          *gorm.DB
	userRepo    repository.UserRepository
	tokens      repository.RefreshTokenRepository
	resetTokens repository.PasswordResetTokenRepository
	mailer      notify.Mailer
	resetURL    string
	resetTTL    time.Duration
	jwtManager  *jwt.Manager
	loginGuard  *ratelimit.FailureLimiter
	// termsVersion is the current terms revision; 0 disables the acceptance gate.
	termsVersion int
	// refreshTTL: sliding window a session extends by on each login/refresh.
	refreshTTL time.Duration
}

// NewAuthService: loginGuard may be nil (no failed-login lockout).
func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, tokens repository.RefreshTokenRepository,
	jwtManager *jwt.Manager, loginGuard *ratelimit.FailureLimiter,
	resetTokens repository.PasswordResetTokenRepository, mailer notify.Mailer,
	resetURL string, resetTTL time.Duration, termsVersion int, refreshTTL time.Duration) AuthService {
	return &authService{db: db, userRepo: userRepo, tokens: tokens, resetTokens: resetTokens,
		mailer: mailer, resetURL: resetURL, resetTTL: resetTTL, jwtManager: jwtManager,
		loginGuard: loginGuard, termsVersion: termsVersion, refreshTTL: refreshTTL}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	email := normalizeEmail(req.Email)

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.ErrEmailAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal("cannot hash password").Wrap(err)
	}

	user := &models.User{
		Email:    email,
		Password: string(hashed),
		FullName: strings.TrimSpace(req.FullName),
		Role:     models.RoleCustomer,
		Active:   true,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if apperrors.IsUniqueViolation(err) {
			return nil, apperrors.ErrEmailAlreadyExists // registered concurrently
		}
		return nil, err
	}
	// Best-effort audit after the fact: a failed audit row must not fail the request.
	s.auditSuccess(ctx, "auth.register", user.ID, user.Role,
		map[string]any{"email": user.Email, "full_name": user.FullName})

	result := dto.NewUserResponse(user)
	return &result, nil
}

// Login counts wrong passwords per email+IP; the 5th in a row locks that pair out.
func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.authenticate(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.termsVersion > 0 && user.AcceptedTermsVersion < s.termsVersion {
		return nil, apperrors.ErrTermsRequired.WithDetails(map[string]string{
			"required_terms_version": strconv.Itoa(s.termsVersion),
			"accepted_terms_version": strconv.Itoa(user.AcceptedTermsVersion),
		})
	}

	pair, err := s.issueTokens(ctx, s.db, user, "", req.DeviceID, req.UserAgent)
	if err != nil {
		return nil, err
	}
	s.auditSuccess(ctx, "auth.login", user.ID, user.Role, nil)

	return &dto.LoginResponse{
		TokenResponse: newTokenResponse(pair),
		User:          dto.NewUserResponse(user),
	}, nil
}

func (s *authService) AcceptTerms(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	if s.termsVersion <= 0 {
		return nil, apperrors.Validation("terms acceptance is not required")
	}
	user, err := s.authenticate(ctx, req)
	if err != nil {
		return nil, err
	}
	if user.AcceptedTermsVersion >= s.termsVersion {
		return nil, apperrors.Validation("terms already accepted")
	}
	if err := s.userRepo.SetAcceptedTerms(ctx, s.db, user.ID, s.termsVersion); err != nil {
		return nil, err
	}
	user.AcceptedTermsVersion = s.termsVersion

	pair, err := s.issueTokens(ctx, s.db, user, "", req.DeviceID, req.UserAgent)
	if err != nil {
		return nil, err
	}
	s.auditSuccess(ctx, "auth.accept_terms", user.ID, user.Role, nil)

	return &dto.LoginResponse{
		TokenResponse: newTokenResponse(pair),
		User:          dto.NewUserResponse(user),
	}, nil
}

func (s *authService) authenticate(ctx context.Context, req dto.LoginRequest) (*models.User, error) {
	email := normalizeEmail(req.Email)
	guardKey := email + "|" + req.ClientIP
	if s.loginGuard != nil {
		if blocked, left := s.loginGuard.Blocked(guardKey); blocked {
			return nil, apperrors.ErrTooManyLoginAttempts.WithDetails(map[string]string{
				"retry_after_seconds": strconv.Itoa(int(math.Ceil(left.Seconds()))),
			})
		}
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists, not even by answering faster.
		if errors.Is(err, apperrors.ErrUserNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(req.Password))
			s.loginFailed(guardKey, req.ClientIP)
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.loginFailed(guardKey, req.ClientIP)
		return nil, apperrors.ErrInvalidCredentials
	}
	if s.loginGuard != nil {
		s.loginGuard.Reset(guardKey)
	}
	if !user.Active {
		return nil, apperrors.ErrAccountLocked
	}
	return user, nil
}

// Refresh rotates the token; a twice-presented token is a replay: whole family revoked, login again.
func (s *authService) Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	claims, err := s.jwtManager.ParseRefresh(refreshToken)
	if err != nil {
		return nil, err
	}

	var (
		pair   *jwt.TokenPair
		user   *models.User
		replay bool
	)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pair, user, replay = nil, nil, false
		stored, err := s.tokens.Lock(ctx, tx, claims.ID)
		if err != nil {
			return err
		}
		if stored == nil || stored.UserID != claims.UserID || stored.RevokedAt != nil {
			return apperrors.ErrInvalidToken
		}
		if stored.UsedAt != nil {
			replay = true
			_, err := s.tokens.RevokeFamily(ctx, tx, stored.FamilyID)
			return err
		}

		u, err := s.userRepo.FindByID(ctx, claims.UserID)
		if err != nil {
			if errors.Is(err, apperrors.ErrUserNotFound) {
				return apperrors.ErrInvalidToken
			}
			return err
		}
		if !u.Active {
			return apperrors.ErrAccountLocked
		}
		if err := s.tokens.MarkUsed(ctx, tx, stored.ID); err != nil {
			return err
		}
		// Device identity carries forward across rotation (client sends device_id/user_agent at login only).
		pair, err = s.issueTokens(ctx, tx, u, stored.FamilyID, stored.DeviceID, stored.UserAgent)
		user = u
		return err
	})
	if err != nil {
		return nil, err
	}
	if replay {
		logger.Warn("refresh token replay detected; token family revoked", logger.String("user_id", claims.UserID))
		return nil, apperrors.ErrInvalidToken
	}
	s.auditSuccess(ctx, "auth.refresh", user.ID, user.Role, nil)

	result := newTokenResponse(pair)
	return &result, nil
}

// Logout revokes the token family; an already-dead token still answers 200.
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.jwtManager.ParseRefresh(refreshToken)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stored, err := s.tokens.Lock(ctx, tx, claims.ID)
		if err != nil {
			return err
		}
		if stored == nil || stored.UserID != claims.UserID || stored.RevokedAt != nil {
			return nil
		}
		_, err = s.tokens.RevokeFamily(ctx, tx, stored.FamilyID)
		return err
	})
}

// ChangePassword checks current password, rotates it, revokes other sessions; current gets fresh pair.
func (s *authService) ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) (*dto.TokenResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal("cannot hash password").Wrap(err)
	}
	var pair *jwt.TokenPair
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.userRepo.SetPassword(ctx, tx, userID, string(hashed)); err != nil {
			return err
		}
		if _, err := s.tokens.RevokeUser(ctx, tx, userID); err != nil {
			return err
		}
		pair, err = s.issueTokens(ctx, tx, user, "", "", "")
		return err
	})
	if err != nil {
		return nil, err
	}
	result := newTokenResponse(pair)
	return &result, nil
}

// ForgotPassword always answers 200 (never reveals existence); missing email burns a bcrypt
// compare so timing doesn't leak.
func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), dummyPasswordHash())
			return nil
		}
		return err
	}
	if !user.Active {
		return nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return apperrors.Internal("cannot generate reset token").Wrap(err)
	}
	tokenHex := hex.EncodeToString(raw)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.resetTokens.InvalidateUser(ctx, tx, user.ID); err != nil {
			return err
		}
		return s.resetTokens.Create(ctx, tx, &models.PasswordResetToken{
			UserID:    user.ID,
			TokenHash: sha256Hex(tokenHex),
			ExpiresAt: time.Now().Add(s.resetTTL),
		})
	})
	if err != nil {
		return err
	}
	if err := s.sendResetEmail(ctx, user.Email, tokenHex); err != nil {
		// The mail is not on the critical path: the client must get its 200 either way.
		logger.Warn("password reset email not sent", logger.String("user_id", user.ID), logger.Err(err))
	}
	return nil
}

// ResetPassword redeems a single-use token (wrong/unused/expired all 400); on success the
// password changes, the token dies and every refresh token is revoked.
func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if len(req.Token) < 16 {
		return apperrors.BadRequest("invalid or expired reset token")
	}
	hash := sha256Hex(strings.ToLower(req.Token))
	var user *models.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		token, err := s.resetTokens.LockValid(ctx, tx, hash)
		if err != nil {
			return err
		}
		if token == nil {
			return apperrors.BadRequest("invalid or expired reset token")
		}
		u, err := s.userRepo.FindByID(ctx, token.UserID)
		if err != nil {
			return err
		}
		if !u.Active {
			return apperrors.ErrAccountLocked
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return apperrors.Internal("cannot hash password").Wrap(err)
		}
		if err := s.userRepo.SetPassword(ctx, tx, u.ID, string(hashed)); err != nil {
			return err
		}
		if err := s.resetTokens.MarkUsed(ctx, tx, token.ID); err != nil {
			return err
		}
		if _, err := s.tokens.RevokeUser(ctx, tx, u.ID); err != nil {
			return err
		}
		user = u
		return nil
	})
	if err != nil {
		return err
	}
	s.auditSuccess(ctx, "auth.reset_password", user.ID, user.Role, nil)
	return nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

const resetEmailHTML = `<!doctype html>
<html><body style="font-family:Arial,Helvetica,sans-serif;color:#1f2328;max-width:480px">
<p>Xin chào,</p>
<p>Bạn vừa yêu cầu đặt lại mật khẩu cho tài khoản xem phim của mình.</p>
<p><a href="{{.URL}}">Đặt lại mật khẩu</a></p>
<p>Liên kết có giá trị 30 phút và chỉ dùng được một lần. Nếu bạn không yêu cầu, hãy bỏ qua email này.</p>
</body></html>`

var resetEmailTemplate = template.Must(template.New("reset").Parse(resetEmailHTML))

// sendResetEmail: the recipient address never enters logs.
func (s *authService) sendResetEmail(ctx context.Context, email, token string) error {
	var body bytes.Buffer
	if err := resetEmailTemplate.Execute(&body, struct{ URL string }{URL: s.resetURL + "?token=" + token}); err != nil {
		return err
	}
	return s.mailer.Send(ctx, notify.Message{To: email, Subject: "Đặt lại mật khẩu", HTML: body.String()})
}

// issueTokens: empty familyID starts a new family (fresh login); each call slides expiry by
// refreshTTL and stamps LastUsedAt, so active devices never silently expire.
func (s *authService) issueTokens(ctx context.Context, tx *gorm.DB, user *models.User, familyID, deviceID, userAgent string) (*jwt.TokenPair, error) {
	pair, err := s.jwtManager.GeneratePair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, apperrors.Internal("cannot issue token").Wrap(err)
	}
	if familyID == "" {
		familyID = uuid.NewString()
	}
	now := time.Now()
	expiresAt := pair.RefreshExpiresAt
	if s.refreshTTL > 0 {
		expiresAt = now.Add(s.refreshTTL)
	}
	if err := s.tokens.Create(ctx, tx, &models.RefreshToken{
		ID:         pair.RefreshID,
		UserID:     user.ID,
		FamilyID:   familyID,
		DeviceID:   deviceID,
		UserAgent:  userAgent,
		ExpiresAt:  expiresAt,
		LastUsedAt: &now,
	}); err != nil {
		return nil, err
	}
	return pair, nil
}

func (s *authService) ListSessions(ctx context.Context, userID, currentDeviceID string) ([]dto.SessionResponse, error) {
	rows, err := s.tokens.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SessionResponse, 0, len(rows))
	for i := range rows {
		row := rows[i]
		out = append(out, dto.SessionResponse{
			ID:         row.ID,
			UserAgent:  row.UserAgent,
			LastUsedAt: row.LastUsedAt,
			CreatedAt:  row.CreatedAt,
			IsCurrent:  currentDeviceID != "" && row.DeviceID == currentDeviceID,
		})
	}
	return out, nil
}

func (s *authService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		affected, err := s.tokens.RevokeByID(ctx, tx, sessionID, userID)
		if err != nil {
			return err
		}
		if affected == 0 {
			return apperrors.ErrSessionNotFound
		}
		return nil
	})
}

// loginFailed logs the IP only: emails stay out of logs.
func (s *authService) loginFailed(key, ip string) {
	if s.loginGuard != nil && s.loginGuard.Fail(key) {
		logger.Warn("login locked out after repeated failures", logger.String("ip", ip))
	}
}

func (s *authService) auditSuccess(ctx context.Context, action, userID, role string, after map[string]any) {
	rec, ok := audit.FromContext(ctx)
	if !ok {
		return
	}
	rec.Action = action
	rec.ActorID = userID
	rec.ActorRole = role
	rec.ResourceType = "user"
	rec.ResourceID = userID
	rec.After = after
	if err := audit.In(ctx, s.db, rec); err != nil {
		logger.L().Warn("auth audit row not written", logger.Err(err), logger.String("action", action))
	}
}

func newTokenResponse(pair *jwt.TokenPair) dto.TokenResponse {
	return dto.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
