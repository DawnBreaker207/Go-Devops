package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"gorm.io/gorm"
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
	jwtManager *jwt.Manager
}

func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, jwtManager *jwt.Manager) AuthService {
	return &authService{db: db, userRepo: userRepo, jwtManager: jwtManager}
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

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		// Don't reveal whether the email exists.
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	pair, err := s.jwtManager.GeneratePair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, apperrors.Internal("cannot issue token").Wrap(err)
	}
	s.auditSuccess(ctx, "auth.login", user.ID, user.Role, nil)

	return &dto.LoginResponse{
		TokenResponse: newTokenResponse(pair),
		User:          dto.NewUserResponse(user),
	}, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	claims, err := s.jwtManager.ParseRefresh(refreshToken)
	if err != nil {
		return nil, err
	}

	// Re-read the user so tokens are invalidated when the account is deleted or demoted.
	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidToken
		}
		return nil, err
	}

	pair, err := s.jwtManager.GeneratePair(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, apperrors.Internal("cannot issue token").Wrap(err)
	}
	s.auditSuccess(ctx, "auth.refresh", user.ID, user.Role, nil)

	result := newTokenResponse(pair)
	return &result, nil
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
