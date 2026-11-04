package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"hr-system/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTypeAccess = "access"

type accessClaims struct {
	Role string `json:"role"`
	Type string `json:"type"`
	jwt.RegisteredClaims
}

func (s *Service) issueTokens(ctx context.Context, user *models.User) (*models.TokenPair, error) {
	now := s.now()

	claims := accessClaims{
		Role: user.RoleCode,
		Type: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
	}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return nil, err
	}

	rt := models.RefreshToken{UserID: user.ID, ExpiresAt: now.Add(s.cfg.RefreshTTL)}
	if err := s.refresh.Create(ctx, hashToken(refresh), rt); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	return &models.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
