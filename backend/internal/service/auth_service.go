package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
)

// AuthService xu ly dang ky, dang nhap va lam moi token.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error)
}

type authService struct {
	userRepo   repository.UserRepository
	jwtManager *jwt.Manager
}

// NewAuthService tao AuthService.
func NewAuthService(userRepo repository.UserRepository, jwtManager *jwt.Manager) AuthService {
	return &authService{userRepo: userRepo, jwtManager: jwtManager}
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

	result := dto.NewUserResponse(user)
	return &result, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		// Khong lo cho client biet email co ton tai hay khong.
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

	// Doc lai user de token bi thu hoi ngay khi tai khoan bi xoa hoac doi quyen.
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

	result := newTokenResponse(pair)
	return &result, nil
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
