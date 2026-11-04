package salary_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/salary"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func newSvc(t *testing.T) (*salary.Service, *mocks.MockSalaryRepository, *mocks.MockEmployeeRepository) {
	t.Helper()
	salaries := mocks.NewMockSalaryRepository(t)
	employees := mocks.NewMockEmployeeRepository(t)
	return salary.NewService(salaries, employees), salaries, employees
}

var date = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

func TestAdd(t *testing.T) {
	ctx := context.Background()
	in := models.SalaryInput{Amount: 5000, EffectiveDate: date}

	t.Run("success", func(t *testing.T) {
		svc, salaries, employees := newSvc(t)
		employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(&models.Employee{ID: 5}, nil).Once()
		salaries.EXPECT().Create(mock.Anything, int64(5), in).Return(&models.Salary{ID: 1, Amount: 5000}, nil).Once()

		got, err := svc.Add(ctx, 5, in)

		require.NoError(t, err)
		require.Equal(t, 5000.0, got.Amount)
	})

	for name, bad := range map[string]models.SalaryInput{
		"zero amount":     {Amount: 0, EffectiveDate: date},
		"negative amount": {Amount: -1, EffectiveDate: date},
		"no date":         {Amount: 100},
	} {
		t.Run(name, func(t *testing.T) {
			svc, _, _ := newSvc(t)

			_, err := svc.Add(ctx, 5, bad)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}

	t.Run("employee not found", func(t *testing.T) {
		svc, _, employees := newSvc(t)
		employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(nil, models.ErrNotFound).Once()

		_, err := svc.Add(ctx, 5, in)

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("same date for the employee", func(t *testing.T) {
		svc, salaries, employees := newSvc(t)
		employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(&models.Employee{ID: 5}, nil).Once()
		salaries.EXPECT().Create(mock.Anything, int64(5), in).Return(nil, models.ErrConflict).Once()

		_, err := svc.Add(ctx, 5, in)

		require.ErrorIs(t, err, models.ErrConflict)
	})
}

func TestList(t *testing.T) {
	ctx := context.Background()

	t.Run("ListByEmployee: success and employee not found", func(t *testing.T) {
		svc, salaries, employees := newSvc(t)
		employees.EXPECT().GetByID(mock.Anything, int64(5)).Return(&models.Employee{ID: 5}, nil).Once()
		employees.EXPECT().GetByID(mock.Anything, int64(9)).Return(nil, models.ErrNotFound).Once()
		salaries.EXPECT().ListByEmployee(mock.Anything, int64(5)).Return([]models.Salary{{ID: 2}, {ID: 1}}, nil).Once()

		items, err := svc.ListByEmployee(ctx, 5)
		require.NoError(t, err)
		require.Len(t, items, 2)

		_, err = svc.ListByEmployee(ctx, 9)
		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("ListMine: own salaries", func(t *testing.T) {
		svc, salaries, employees := newSvc(t)
		employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(&models.Employee{ID: 5}, nil).Once()
		salaries.EXPECT().ListByEmployee(mock.Anything, int64(5)).Return([]models.Salary{{ID: 1}}, nil).Once()

		items, err := svc.ListMine(ctx, models.CurrentUser{ID: 50, Role: "employee"})

		require.NoError(t, err)
		require.Len(t, items, 1)
	})

	t.Run("ListMine: HR without an employee row gets an empty list", func(t *testing.T) {
		svc, _, employees := newSvc(t)
		employees.EXPECT().GetByUserID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()

		items, err := svc.ListMine(ctx, models.CurrentUser{ID: 1, Role: "hr"})

		require.NoError(t, err)
		require.NotNil(t, items)
		require.Empty(t, items)
	})

	t.Run("ListMine: employee lookup failure", func(t *testing.T) {
		svc, _, employees := newSvc(t)
		employees.EXPECT().GetByUserID(mock.Anything, int64(1)).Return(nil, errBoom).Once()

		_, err := svc.ListMine(ctx, models.CurrentUser{ID: 1, Role: "hr"})

		require.ErrorIs(t, err, errBoom)
	})
}
