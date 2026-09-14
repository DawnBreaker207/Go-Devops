// Package jwt issues and validates access/refresh token pairs.
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// TokenType distinguishes access and refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the token payload.
type Claims struct {
	UserID string    `json:"uid"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Type   TokenType `json:"typ"`
	jwtlib.RegisteredClaims
}

// TokenPair is the token pair returned to the client. RefreshID and
// RefreshExpiresAt let the caller persist the refresh token for rotation.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshID        string
	RefreshExpiresAt time.Time
}

// Manager holds the secrets and TTLs for each token type.
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

func NewManager(accessSecret, refreshSecret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		issuer:        issuer,
	}
}

// GeneratePair issues an access and refresh token pair.
func (m *Manager) GeneratePair(userID, email, role string) (*TokenPair, error) {
	accessToken, _, _, err := m.sign(userID, email, role, AccessToken, m.accessSecret, m.accessTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshID, refreshExp, err := m.sign(userID, email, role, RefreshToken, m.refreshSecret, m.refreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(m.accessTTL.Seconds()),
		RefreshID:        refreshID,
		RefreshExpiresAt: refreshExp,
	}, nil
}

// ParseAccess validates an access token.
func (m *Manager) ParseAccess(token string) (*Claims, error) {
	return m.parse(token, m.accessSecret, AccessToken)
}

// ParseRefresh validates a refresh token.
func (m *Manager) ParseRefresh(token string) (*Claims, error) {
	return m.parse(token, m.refreshSecret, RefreshToken)
}

// sign returns the token with its jti and expiry.
func (m *Manager) sign(userID, email, role string, tokenType TokenType, secret []byte, ttl time.Duration) (string, string, time.Time, error) {
	now := time.Now()
	id := uuid.NewString()
	expires := now.Add(ttl)
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   tokenType,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ID:        id,
			Subject:   userID,
			Issuer:    m.issuer,
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(expires),
		},
	}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString(secret)
	return token, id, expires, err
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
