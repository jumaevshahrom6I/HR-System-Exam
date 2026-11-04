package auth_test

import (
	"context"
	"testing"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func accessFor(t *testing.T, ttl time.Duration) string {
	t.Helper()
	e := newEnv(t, ttl)
	e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(newUser(t, 7, "hr@company.local", "Pass12345", "hr", true), nil).Once()
	e.refresh.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	pair, err := e.svc.Login(context.Background(), models.LoginInput{Email: "hr@company.local", Password: "Pass12345"})
	require.NoError(t, err)
	return pair.AccessToken
}

func forge(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return token
}

func TestAuthenticate(t *testing.T) {
	svc := newEnv(t, 15*time.Minute).svc
	exp := time.Now().Add(time.Hour).Unix()

	t.Run("valid token: id and role", func(t *testing.T) {
		claims, err := svc.Authenticate(accessFor(t, 15*time.Minute))

		require.NoError(t, err)
		require.Equal(t, int64(7), claims.UserID)
		require.Equal(t, "hr", claims.Role)
	})

	tests := []struct {
		name  string
		token string
	}{
		{"garbage instead of a token", "not-a-jwt"},
		{"empty string", ""},
		{"foreign signature", forge(t, "another-secret-another-secret-xx", jwt.MapClaims{"sub": "7", "role": "hr", "type": "access", "exp": exp})},
		{"expired token", accessFor(t, -time.Minute)},
		{"token type is not access", forge(t, testSecret, jwt.MapClaims{"sub": "7", "role": "hr", "type": "refresh", "exp": exp})},
		{"no role", forge(t, testSecret, jwt.MapClaims{"sub": "7", "type": "access", "exp": exp})},
		{"sub is not a number", forge(t, testSecret, jwt.MapClaims{"sub": "abc", "role": "hr", "type": "access", "exp": exp})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Authenticate(tt.token)

			require.ErrorIs(t, err, models.ErrInvalidToken)
		})
	}

	t.Run("a service with a different secret rejects the token", func(t *testing.T) {
		other := auth.NewService(mocks.NewMockUserRepository(t), mocks.NewMockRefreshTokenRepository(t),
			auth.Config{Secret: "different-secret-different-secret", AccessTTL: time.Hour, RefreshTTL: time.Hour})

		_, err := other.Authenticate(accessFor(t, time.Hour))

		require.ErrorIs(t, err, models.ErrInvalidToken)
	})
}
