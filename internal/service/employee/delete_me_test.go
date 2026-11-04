package employee_test

import (
	"context"
	"testing"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("employee not found", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(999)).Return(nil, models.ErrNotFound).Once()

		require.ErrorIs(t, e.svc.Delete(ctx, 999), models.ErrNotFound)
	})

	t.Run("success: terminated, tokens revoked for the account", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.employees.EXPECT().Terminate(mock.Anything, int64(5)).Return(nil).Once()
		e.refresh.EXPECT().RevokeAllForUser(mock.Anything, int64(50)).Return(nil).Once()

		require.NoError(t, e.svc.Delete(ctx, 5))
	})

	t.Run("already terminated", func(t *testing.T) {
		e := newEnv(t)
		emp := activeEmployee()
		emp.Status = "terminated"
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(emp, nil).Once()

		require.ErrorIs(t, e.svc.Delete(ctx, 5), models.ErrConflict)
	})

	t.Run("termination failure: tokens are not revoked", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.employees.EXPECT().Terminate(mock.Anything, int64(5)).Return(errBoom).Once()

		require.ErrorIs(t, e.svc.Delete(ctx, 5), errBoom)
	})

	t.Run("token revocation failure is returned as an error", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.employees.EXPECT().Terminate(mock.Anything, int64(5)).Return(nil).Once()
		e.refresh.EXPECT().RevokeAllForUser(mock.Anything, int64(50)).Return(errBoom).Once()

		require.ErrorIs(t, e.svc.Delete(ctx, 5), errBoom)
	})
}

func TestMe(t *testing.T) {
	ctx := context.Background()

	t.Run("employee: profile with the employee row", func(t *testing.T) {
		e := newEnv(t)
		e.users.EXPECT().GetByID(mock.Anything, int64(50)).Return(&models.User{ID: 50, Email: "ivan@company.local", RoleCode: "employee"}, nil).Once()
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(activeEmployee(), nil).Once()

		p, err := e.svc.Me(ctx, models.CurrentUser{ID: 50, Role: "employee"})

		require.NoError(t, err)
		require.Equal(t, "ivan@company.local", p.Email)
		require.NotNil(t, p.Employee)
	})

	t.Run("HR without an employees row: not an error, employee = nil", func(t *testing.T) {
		e := newEnv(t)
		e.users.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.User{ID: 1, Email: "hr@company.local", RoleCode: "hr"}, nil).Once()
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()

		p, err := e.svc.Me(ctx, models.CurrentUser{ID: 1, Role: "hr"})

		require.NoError(t, err)
		require.Nil(t, p.Employee)
		require.Equal(t, "hr", p.Role)
	})

	t.Run("account not found", func(t *testing.T) {
		e := newEnv(t)
		e.users.EXPECT().GetByID(mock.Anything, int64(9)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Me(ctx, models.CurrentUser{ID: 9, Role: "hr"})

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("failure while reading the employee", func(t *testing.T) {
		e := newEnv(t)
		e.users.EXPECT().GetByID(mock.Anything, int64(50)).Return(&models.User{ID: 50}, nil).Once()
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(nil, errBoom).Once()

		_, err := e.svc.Me(ctx, models.CurrentUser{ID: 50, Role: "employee"})

		require.ErrorIs(t, err, errBoom)
	})
}
