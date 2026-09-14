package service

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

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

// AuthService handles register, login, and token refresh.
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

// NewAuthService wires auth. loginGuard may be nil (no failed-login lockout).
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
		return nil, err
	}
	// Auth events are activity records, not handled by the middleware's
	// failure path; written after the fact, best-effort — a failed audit row
	// must not take the user's session down.
	s.auditSuccess(ctx, "auth.register", user.ID, user.Role,
		map[string]any{"email": user.Email, "full_name": user.FullName})

	result := dto.NewUserResponse(user)
	return &result, nil
}

// Login checks the password, then the lockout state of the account. Wrong
// passwords count per email+IP; the 5th in a row locks that pair out (T14).
func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	email := normalizeEmail(req.Email)
	guardKey := email + "|" + req.ClientIP
	if s.loginGuard != nil {
		if blocked, left := s.loginGuard.Blocked(guardKey); blocked {
			// FR-AUTH-03: tell the client how long to wait.
			return nil, apperrors.ErrTooManyLoginAttempts.WithDetails(map[string]string{
				"retry_after_seconds": strconv.Itoa(int(math.Ceil(left.Seconds()))),
			})
		}
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal whether the email exists.
		if errors.Is(err, apperrors.ErrUserNotFound) {
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
		return nil, apperrors.ErrAccountLocked // E-U3
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

// Refresh rotates the refresh token: the presented token is consumed and a
// new one issued in the same family. A token presented twice is a replay:
// the whole family is revoked and the user must log in again (E-C4, T13).
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

// issueTokens signs a pair and stores its refresh token; an empty familyID
// starts a new family (a fresh login).
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

// loginFailed counts a failure. The log carries the IP only: emails stay out
// of logs (NFR-LEG-01).
func (s *authService) loginFailed(key, ip string) {
	if s.loginGuard != nil && s.loginGuard.Fail(key) {
		logger.Warn("login locked out after repeated failures", logger.String("ip", ip))
	}
}

// auditSuccess writes an auth event row. The middleware stashes the route's
// action and network info into the context; here we only complete it.
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
