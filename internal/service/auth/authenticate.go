package auth

import (
	"strconv"

	"hr-system/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

func (s *Service) Authenticate(accessToken string) (*models.AccessClaims, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(accessToken, claims,
		func(*jwt.Token) (any, error) { return []byte(s.cfg.Secret), nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil || !token.Valid {
		return nil, models.ErrInvalidToken
	}

	if claims.Type != tokenTypeAccess {
		return nil, models.ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 || claims.Role == "" {
		return nil, models.ErrInvalidToken
	}

	return &models.AccessClaims{UserID: userID, Role: claims.Role}, nil
}
