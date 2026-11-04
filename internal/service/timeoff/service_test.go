package timeoff_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/timeoff"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

type env struct {
	svc       *timeoff.Service
	timeoffs  *mocks.MockTimeOffRepository
	employees *mocks.MockEmployeeRepository
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{timeoffs: mocks.NewMockTimeOffRepository(t), employees: mocks.NewMockEmployeeRepository(t)}
	e.svc = timeoff.NewService(e.timeoffs, e.employees)
	return e
}

var (
	employeeUser = models.CurrentUser{ID: 50, Role: "employee"}
	hrUser       = models.CurrentUser{ID: 1, Role: "hr"}
	day          = func(d int) time.Time { return time.Date(2026, 7, d, 0, 0, 0, 0, time.UTC) }
)

func validInput() models.TimeOffInput {
	return models.TimeOffInput{LeaveType: "vacation", StartDate: day(1), EndDate: day(14)}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("success: employee_id comes from the token", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(&models.Employee{ID: 5, UserID: 50}, nil).Once()
		e.timeoffs.EXPECT().Create(mock.Anything, int64(5), validInput()).Return(&models.TimeOff{ID: 1, EmployeeID: 5, Status: "pending"}, nil).Once()

		got, err := e.svc.Create(ctx, employeeUser, validInput())

		require.NoError(t, err)
		require.Equal(t, "pending", got.Status)
	})

	invalid := []struct {
		name   string
		mutate func(in *models.TimeOffInput)
	}{
		{"unknown type", func(in *models.TimeOffInput) { in.LeaveType = "weekend" }},
		{"end before start", func(in *models.TimeOffInput) { in.EndDate = day(1).AddDate(0, 0, -1) }},
		{"no dates", func(in *models.TimeOffInput) { in.StartDate = time.Time{} }},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			in := validInput()
			tt.mutate(&in)

			_, err := e.svc.Create(ctx, employeeUser, in)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}

	t.Run("user without an employee row", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.Create(ctx, hrUser, validInput())

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("overlaps with another request", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(&models.Employee{ID: 5}, nil).Once()
		e.timeoffs.EXPECT().Create(mock.Anything, int64(5), mock.Anything).Return(nil, models.ErrConflict).Once()

		_, err := e.svc.Create(ctx, employeeUser, validInput())

		require.ErrorIs(t, err, models.ErrConflict)
	})

	t.Run("employee lookup failure", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(nil, errBoom).Once()

		_, err := e.svc.Create(ctx, employeeUser, validInput())

		require.ErrorIs(t, err, errBoom)
	})
}

func TestList(t *testing.T) {
	ctx := context.Background()
	other := int64(77)

	t.Run("HR sees all and can filter", func(t *testing.T) {
		e := newEnv(t)
		e.timeoffs.EXPECT().List(mock.Anything, models.TimeOffFilter{Status: "pending", EmployeeID: &other}).Return([]models.TimeOff{{ID: 1}}, nil).Once()

		items, err := e.svc.List(ctx, hrUser, models.TimeOffFilter{Status: "pending", EmployeeID: &other})

		require.NoError(t, err)
		require.Len(t, items, 1)
	})

	t.Run("employee sees only own: another employee_id is ignored", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(&models.Employee{ID: 5}, nil).Once()
		e.timeoffs.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.TimeOffFilter) bool {
			return f.EmployeeID != nil && *f.EmployeeID == 5
		})).Return([]models.TimeOff{}, nil).Once()

		_, err := e.svc.List(ctx, employeeUser, models.TimeOffFilter{EmployeeID: &other})

		require.NoError(t, err)
	})

	t.Run("no employee row: empty list", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().GetByUserID(mock.Anything, int64(50)).Return(nil, models.ErrNotFound).Once()

		items, err := e.svc.List(ctx, employeeUser, models.TimeOffFilter{})

		require.NoError(t, err)
		require.Empty(t, items)
		require.NotNil(t, items)
	})

	t.Run("unknown status in filter", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.List(ctx, hrUser, models.TimeOffFilter{Status: "weird"})

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("HR approves: approved_by is the HR user id", func(t *testing.T) {
		e := newEnv(t)
		e.timeoffs.EXPECT().GetByID(mock.Anything, int64(3)).Return(&models.TimeOff{ID: 3, Status: "pending"}, nil).Once()
		e.timeoffs.EXPECT().Decide(mock.Anything, int64(3), "approved", int64(1)).Return(&models.TimeOff{ID: 3, Status: "approved"}, nil).Once()

		got, err := e.svc.UpdateStatus(ctx, hrUser, 3, "approved")

		require.NoError(t, err)
		require.Equal(t, "approved", got.Status)
	})

	t.Run("employee cannot change status", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UpdateStatus(ctx, employeeUser, 3, "approved")

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("status pending is not valid as a decision", func(t *testing.T) {
		e := newEnv(t)

		_, err := e.svc.UpdateStatus(ctx, hrUser, 3, "pending")

		require.ErrorIs(t, err, models.ErrInvalidInput)
	})

	t.Run("request not found", func(t *testing.T) {
		e := newEnv(t)
		e.timeoffs.EXPECT().GetByID(mock.Anything, int64(3)).Return(nil, models.ErrNotFound).Once()

		_, err := e.svc.UpdateStatus(ctx, hrUser, 3, "rejected")

		require.ErrorIs(t, err, models.ErrNotFound)
	})

	t.Run("repeated decision: request is no longer pending", func(t *testing.T) {
		e := newEnv(t)
		e.timeoffs.EXPECT().GetByID(mock.Anything, int64(3)).Return(&models.TimeOff{ID: 3, Status: "approved"}, nil).Once()

		_, err := e.svc.UpdateStatus(ctx, hrUser, 3, "rejected")

		require.ErrorIs(t, err, models.ErrConflict)
	})

	t.Run("race: status changed between check and write", func(t *testing.T) {
		e := newEnv(t)
		e.timeoffs.EXPECT().GetByID(mock.Anything, int64(3)).Return(&models.TimeOff{ID: 3, Status: "pending"}, nil).Once()
		e.timeoffs.EXPECT().Decide(mock.Anything, int64(3), "approved", int64(1)).Return(nil, models.ErrConflict).Once()

		_, err := e.svc.UpdateStatus(ctx, hrUser, 3, "approved")

		require.ErrorIs(t, err, models.ErrConflict)
	})
}
