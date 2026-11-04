package employee_test

import (
	"errors"
	"testing"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository/mocks"
	"hr-system/internal/service/employee"
)

var errBoom = errors.New("boom")

type env struct {
	svc         *employee.Service
	employees   *mocks.MockEmployeeRepository
	users       *mocks.MockUserRepository
	roles       *mocks.MockRoleRepository
	departments *mocks.MockDepartmentRepository
	positions   *mocks.MockPositionRepository
	refresh     *mocks.MockRefreshTokenRepository
	history     *mocks.MockHistoryRepository
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		employees:   mocks.NewMockEmployeeRepository(t),
		users:       mocks.NewMockUserRepository(t),
		roles:       mocks.NewMockRoleRepository(t),
		departments: mocks.NewMockDepartmentRepository(t),
		positions:   mocks.NewMockPositionRepository(t),
		refresh:     mocks.NewMockRefreshTokenRepository(t),
		history:     mocks.NewMockHistoryRepository(t),
	}
	e.svc = employee.NewService(e.employees, e.users, e.roles, e.departments, e.positions, e.refresh, e.history)
	return e
}

func validInput() models.EmployeeInput {
	return models.EmployeeInput{
		Email: "ivan@company.local", Password: "Pass12345",
		FirstName: "Ivan", LastName: "Petrov", Phone: "+992900000000",
		DepartmentID: 1, PositionID: 1, HireDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	}
}

func activeEmployee() *models.Employee {
	return &models.Employee{ID: 5, UserID: 50, Email: "ivan@company.local", FirstName: "Ivan", LastName: "Petrov",
		Phone: "+992900000000", DepartmentID: 1, DepartmentName: "IT", PositionID: 1, PositionName: "Dev",
		HireDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Status: "active"}
}
