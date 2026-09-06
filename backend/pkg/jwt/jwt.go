// Package jwt phat hanh va xac thuc cap access/refresh token.
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// TokenType phan biet access va refresh token de khong dung lan nhau.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims la payload nhung trong token.
type Claims struct {
	UserID string    `json:"uid"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Type   TokenType `json:"typ"`
	jwtlib.RegisteredClaims
}

// TokenPair la cap token tra ve cho client.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// Manager giu secret va thoi han cua tung loai token.
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

// NewManager tao manager tu cau hinh.
func NewManager(accessSecret, refreshSecret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}
}

// GeneratePair phat hanh dong thoi access token va refresh token.
func (m *Manager) GeneratePair(userID, email, role string) (*TokenPair, error) {
	accessToken, err := m.sign(userID, email, role, AccessToken, m.accessSecret, m.accessTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.sign(userID, email, role, RefreshToken, m.refreshSecret, m.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.accessTTL.Seconds()),
	}, nil
}

// ParseAccess xac thuc access token.
func (m *Manager) ParseAccess(token string) (*Claims, error) {
	return m.parse(token, m.accessSecret, AccessToken)
}

// ParseRefresh xac thuc refresh token.
func (m *Manager) ParseRefresh(token string) (*Claims, error) {
	return m.parse(token, m.refreshSecret, RefreshToken)
}

func (m *Manager) sign(userID, email, role string, tokenType TokenType, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(secret)
}

func (m *Manager) parse(token string, secret []byte, expected TokenType) (*Claims, error) {
	parsed, err := jwtlib.ParseWithClaims(
		token,
		&Claims{},
		func(t *jwtlib.Token) (any, error) {
			if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return nil, apperrors.TokenExpired("token has expired").Wrap(err)
		}
		return nil, apperrors.ErrInvalidToken.Wrap(err)
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, apperrors.ErrInvalidToken
	}
	if claims.Type != expected {
		return nil, apperrors.ErrInvalidToken
	}
	return claims, nil
}
