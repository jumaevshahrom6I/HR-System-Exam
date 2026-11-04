package employee_test

import (
	"context"
	"strings"
	"testing"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func expectRefsExist(e *env) {
	e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Department{ID: 1, Name: "IT"}, nil).Once()
	e.positions.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Position{ID: 1, Name: "Dev"}, nil).Once()
}

func TestCreate_Success(t *testing.T) {
	e := newEnv(t)
	expectRefsExist(e)
	e.roles.EXPECT().GetByCode(mock.Anything, "employee").Return(&models.Role{ID: 2, Code: "employee"}, nil).Once()
	e.employees.EXPECT().Create(
		mock.Anything,
		mock.MatchedBy(func(in models.EmployeeInput) bool { return in.Email == "ivan@company.local" }),
		mock.MatchedBy(func(hash string) bool {
			return hash != "Pass12345" && bcrypt.CompareHashAndPassword([]byte(hash), []byte("Pass12345")) == nil
		}),
		int64(2),
	).Return(activeEmployee(), nil).Once()
	in := validInput()
	in.Email = "  Ivan@Company.Local "

	got, err := e.svc.Create(context.Background(), in)

	require.NoError(t, err)
	require.Equal(t, int64(5), got.ID)
}

func TestCreate_InvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(in *models.EmployeeInput)
	}{
		{"invalid email", func(in *models.EmployeeInput) { in.Email = "not-an-email" }},
		{"empty email", func(in *models.EmployeeInput) { in.Email = "" }},
		{"password of 7 bytes", func(in *models.EmployeeInput) { in.Password = "1234567" }},
		{"password of 73 bytes", func(in *models.EmployeeInput) { in.Password = strings.Repeat("a", 73) }},
		{"empty name", func(in *models.EmployeeInput) { in.FirstName = "" }},
		{"name of spaces only", func(in *models.EmployeeInput) { in.FirstName = "   " }},
		{"empty phone", func(in *models.EmployeeInput) { in.Phone = "" }},
		{"phone of 21 characters", func(in *models.EmployeeInput) { in.Phone = strings.Repeat("1", 21) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			in := validInput()
			tt.mutate(&in)

			_, err := e.svc.Create(context.Background(), in)

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}
}

func TestCreate_Dependencies(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(e *env)
		wantErr error
	}{
		{
			name: "department does not exist",
			setup: func(e *env) {
				e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()
			},
			wantErr: models.ErrInvalidInput,
		},
		{
			name: "position does not exist",
			setup: func(e *env) {
				e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Department{ID: 1}, nil).Once()
				e.positions.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, models.ErrNotFound).Once()
			},
			wantErr: models.ErrInvalidInput,
		},
		{
			name: "failure while checking the department",
			setup: func(e *env) {
				e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "email already taken",
			setup: func(e *env) {
				expectRefsExist(e)
				e.roles.EXPECT().GetByCode(mock.Anything, "employee").Return(&models.Role{ID: 2}, nil).Once()
				e.employees.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, int64(2)).Return(nil, models.ErrConflict).Once()
			},
			wantErr: models.ErrConflict,
		},
		{
			name: "other repository error is returned wrapped",
			setup: func(e *env) {
				expectRefsExist(e)
				e.roles.EXPECT().GetByCode(mock.Anything, "employee").Return(&models.Role{ID: 2}, nil).Once()
				e.employees.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything, int64(2)).Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "role employee not found",
			setup: func(e *env) {
				expectRefsExist(e)
				e.roles.EXPECT().GetByCode(mock.Anything, "employee").Return(nil, models.ErrNotFound).Once()
			},
			wantErr: models.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			tt.setup(e)

			_, err := e.svc.Create(context.Background(), validInput())

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
