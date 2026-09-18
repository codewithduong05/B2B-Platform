package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrTokenRevoked     = errors.New("token has been revoked")
)

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID      int64  `json:"user_id"`
	UserCode    string `json:"user_code"`
	Email       string `json:"email"`
	UserType    string `json:"user_type"`
	TokenType   string `json:"token_type"` // "access" or "refresh"
	TokenFamily string `json:"token_family,omitempty"`
}

type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
	audience        string
}

func NewJWTManager(cfg *config.JWTConfig) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(cfg.SecretKey),
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		issuer:          cfg.Issuer,
		audience:        cfg.Audience,
	}
}

func (m *JWTManager) GenerateAccessToken(userID int64, userCode, email, userType string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenTTL)

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		UserCode:  userCode,
		Email:     email,
		UserType:  userType,
		TokenType: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func (m *JWTManager) GenerateRefreshToken(userID int64, userCode string, family string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTokenTTL)

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:      userID,
		UserCode:    userCode,
		TokenType:   "refresh",
		TokenFamily: family,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return tokenString, expiresAt, nil
}

func (m *JWTManager) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) ValidateRefreshToken(tokenString string) (*TokenClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

type TokenPair struct {
	AccessToken   string
	RefreshToken  string
	AccessExpiry  time.Time
	RefreshExpiry time.Time
}

func (m *JWTManager) GenerateTokenPair(userID int64, userCode, email, userType string, family string) (*TokenPair, error) {
	accessToken, accessExpiry, err := m.GenerateAccessToken(userID, userCode, email, userType)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExpiry, err := m.GenerateRefreshToken(userID, userCode, family)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}, nil
}

func (m *JWTManager) RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := m.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// Generate new token pair with same family
	return m.GenerateTokenPair(claims.UserID, claims.UserCode, claims.Email, claims.UserType, claims.TokenFamily)
}

// TokenRevoker is an interface for checking if a token has been revoked
type TokenRevoker interface {
	IsRevoked(ctx context.Context, tokenID string) (bool, error)
	Revoke(ctx context.Context, tokenID string, reason string) error
	RevokeFamily(ctx context.Context, familyID string, reason string) error
}
