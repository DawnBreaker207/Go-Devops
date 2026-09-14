package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
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
	Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)
}

type authService struct {
	db         *gorm.DB
	userRepo   repository.UserRepository
	tokens     repository.RefreshTokenRepository
	jwtManager *jwt.Manager
	loginGuard *ratelimit.FailureLimiter
}

// NewAuthService: loginGuard may be nil (no failed-login lockout).
func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, tokens repository.RefreshTokenRepository,
	jwtManager *jwt.Manager, loginGuard *ratelimit.FailureLimiter) AuthService {
	return &authService{db: db, userRepo: userRepo, tokens: tokens, jwtManager: jwtManager, loginGuard: loginGuard}
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

	pair, err := s.issueTokens(ctx, s.db, user, "")
	if err != nil {
		return nil, err
	}
	s.auditSuccess(ctx, "auth.login", user.ID, user.Role, nil)

	return &dto.LoginResponse{
		TokenResponse: newTokenResponse(pair),
		User:          dto.NewUserResponse(user),
	}, nil
}

// Refresh rotates the refresh token. A token presented twice is a replay: its whole
// family is revoked and the user must log in again.
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
		pair, err = s.issueTokens(ctx, tx, u, stored.FamilyID)
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

// issueTokens: an empty familyID starts a new token family (a fresh login).
func (s *authService) issueTokens(ctx context.Context, tx *gorm.DB, user *models.User, familyID string) (*jwt.TokenPair, error) {
	pair, err := s.jwtManager.GeneratePair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, apperrors.Internal("cannot issue token").Wrap(err)
	}
	if familyID == "" {
		familyID = uuid.NewString()
	}
	if err := s.tokens.Create(ctx, tx, &models.RefreshToken{
		ID:        pair.RefreshID,
		UserID:    user.ID,
		FamilyID:  familyID,
		ExpiresAt: pair.RefreshExpiresAt,
	}); err != nil {
		return nil, err
	}
	return pair, nil
}

// loginFailed logs the IP only: emails stay out of logs.
func (s *authService) loginFailed(key, ip string) {
	if s.loginGuard != nil && s.loginGuard.Fail(key) {
		logger.Warn("login locked out after repeated failures", logger.String("ip", ip))
	}
}

// auditSuccess completes the audit record the middleware put in the context.
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
