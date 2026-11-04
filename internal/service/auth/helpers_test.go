package auth_test

import (
	"testing"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/auth"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret-test-secret-test-secret"

type env struct {
	svc     *auth.Service
	users   *mocks.MockUserRepository
	refresh *mocks.MockRefreshTokenRepository
}

func newEnv(t *testing.T, accessTTL time.Duration) *env {
	t.Helper()
	users := mocks.NewMockUserRepository(t)
	refresh := mocks.NewMockRefreshTokenRepository(t)
	svc := auth.NewService(users, refresh, auth.Config{Secret: testSecret, AccessTTL: accessTTL, RefreshTTL: time.Hour})
	return &env{svc: svc, users: users, refresh: refresh}
}

func newUser(t *testing.T, id int64, email, password, role string, active bool) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return &models.User{ID: id, RoleID: 1, RoleCode: role, Email: email, PasswordHash: string(hash), IsActive: active}
}
