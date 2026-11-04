package auth_test

import (
	"context"
	"testing"
	"time"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRefresh(t *testing.T) {
	ctx := context.Background()

	t.Run("success: new pair issued and new refresh saved", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.AnythingOfType("string")).Return(int64(1), nil).Once()
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
		e.refresh.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("models.RefreshToken")).Return(nil).Once()

		pair, err := e.svc.Refresh(ctx, "old-token")

		require.NoError(t, err)
		require.NotEmpty(t, pair.AccessToken)
		require.NotEmpty(t, pair.RefreshToken)
		require.NotEqual(t, "old-token", pair.RefreshToken)
	})

	t.Run("reuse: second call is rejected", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(0), models.ErrNotFound).Once()
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
		e.refresh.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		_, err := e.svc.Refresh(ctx, "token")
		require.NoError(t, err)

		_, err = e.svc.Refresh(ctx, "token")
		require.ErrorIs(t, err, models.ErrInvalidToken)
	})

	t.Run("token is unknown or expired", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(0), models.ErrNotFound).Once()

		_, err := e.svc.Refresh(ctx, "no-such-token")

		require.ErrorIs(t, err, models.ErrInvalidToken)
	})

	t.Run("empty token: repositories are not called", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)

		_, err := e.svc.Refresh(ctx, "")

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})

	t.Run("Redis failure is returned as an error", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(0), errBoom).Once()

		_, err := e.svc.Refresh(ctx, "token")

		require.ErrorIs(t, err, errBoom)
		require.NotErrorIs(t, err, models.ErrInvalidToken)
	})

	t.Run("GetByID failure after Consume: no new pair issued", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errBoom).Once()

		pair, err := e.svc.Refresh(ctx, "token")

		require.ErrorIs(t, err, errBoom)
		require.Nil(t, pair)
	})

	t.Run("token owner not found", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Refresh(ctx, "token")

		require.ErrorIs(t, err, models.ErrInvalidToken)
	})

	t.Run("account deactivated after the token was issued", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Consume(mock.Anything, mock.Anything).Return(int64(1), nil).Once()
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", false), nil).Once()

		_, err := e.svc.Refresh(ctx, "token")

		require.ErrorIs(t, err, models.ErrInvalidToken)
	})
}

func TestLogout(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Delete(mock.Anything, mock.AnythingOfType("string")).Return(nil).Once()

		require.NoError(t, e.svc.Logout(ctx, "token"))
	})

	t.Run("Redis failure", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)
		e.refresh.EXPECT().Delete(mock.Anything, mock.Anything).Return(errBoom).Once()

		require.ErrorIs(t, e.svc.Logout(ctx, "token"), errBoom)
	})

	t.Run("empty token", func(t *testing.T) {
		e := newEnv(t, 15*time.Minute)

		require.ErrorIs(t, e.svc.Logout(ctx, ""), models.ErrInvalidInput)
	})
}
