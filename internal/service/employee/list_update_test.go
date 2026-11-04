package employee_test

import (
	"context"
	"testing"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	invalid := []struct {
		name   string
		filter models.EmployeeFilter
	}{
		{"limit 1000", models.EmployeeFilter{Page: 1, Limit: 1000}},
		{"limit 101", models.EmployeeFilter{Page: 1, Limit: 101}},
		{"limit 0", models.EmployeeFilter{Page: 1, Limit: 0}},
		{"page 0", models.EmployeeFilter{Page: 0, Limit: 20}},
		{"unknown status", models.EmployeeFilter{Page: 1, Limit: 20, Status: "weird"}},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			_, err := e.svc.List(context.Background(), tt.filter)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}

	t.Run("success: page and total", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.EmployeeFilter) bool {
			return f.Q == "iv" && f.Page == 2 && f.Limit == 100 && f.Status == "active"
		})).Return([]models.Employee{*activeEmployee()}, int64(101), nil).Once()

		page, err := e.svc.List(context.Background(), models.EmployeeFilter{Q: "  iv ", Page: 2, Limit: 100, Status: "active"})

		require.NoError(t, err)
		require.Equal(t, int64(101), page.Total)
		require.Equal(t, 2, page.Page)
		require.Len(t, page.Items, 1)
	})

	t.Run("repository failure", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().List(mock.Anything, mock.Anything).Return(nil, int64(0), errBoom).Once()

		_, err := e.svc.List(context.Background(), models.EmployeeFilter{Page: 1, Limit: 20})

		require.ErrorIs(t, err, errBoom)
	})
}

func validUpdate() models.EmployeeUpdate {
	return models.EmployeeUpdate{FirstName: "Ivan", LastName: "Petrov", Phone: "+992900000000", DepartmentID: 1, PositionID: 1, Status: "active"}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		expectRefsExist(e)
		e.employees.EXPECT().Update(mock.Anything, int64(5), mock.Anything).Return(activeEmployee(), nil).Once()

		got, err := e.svc.Update(ctx, 5, validUpdate())

		require.NoError(t, err)
		require.Equal(t, int64(5), got.ID)
	})

	t.Run("status terminated via PUT is not allowed", func(t *testing.T) {
		e := newEnv(t)
		in := validUpdate()
		in.Status = "terminated"

		_, err := e.svc.Update(ctx, 5, in)

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})

	t.Run("a terminated employee cannot be changed", func(t *testing.T) {
		e := newEnv(t)
		emp := activeEmployee()
		emp.Status = "terminated"
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(emp, nil).Once()

		_, err := e.svc.Update(ctx, 5, validUpdate())

		require.ErrorIs(t, err, models.ErrConflict)
	})

	t.Run("employee not found", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(999)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Update(ctx, 999, validUpdate())

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("new department does not exist", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Update(ctx, 5, validUpdate())

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})

	t.Run("invalid data: repositories are not called", func(t *testing.T) {
		e := newEnv(t)
		in := validUpdate()
		in.Status = "weird"

		_, err := e.svc.Update(ctx, 5, in)

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})
}

func TestGetAndHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("Get: not found", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(999)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Get(ctx, 999)

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("History: employee not found", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(999)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.History(ctx, 999)

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("History: success", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.history.EXPECT().ListByEmployee(mock.Anything, int64(5)).Return([]models.PositionHistory{{ID: 2}, {ID: 1}}, nil).Once()

		items, err := e.svc.History(ctx, 5)

		require.NoError(t, err)
		require.Len(t, items, 2)
	})

	t.Run("History: history repository failure", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(activeEmployee(), nil).Once()
		e.history.EXPECT().ListByEmployee(mock.Anything, int64(5)).Return(nil, errBoom).Once()

		_, err := e.svc.History(ctx, 5)

		require.ErrorIs(t, err, errBoom)
	})
}
