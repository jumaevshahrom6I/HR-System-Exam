package repository

import (
	"context"

	"hr-system/internal/models"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	GetByID(ctx context.Context, id int64) (*models.User, error)
}

type RoleRepository interface {
	GetByCode(ctx context.Context, code string) (*models.Role, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, tokenHash string, rt models.RefreshToken) error

	Consume(ctx context.Context, tokenHash string) (int64, error)

	Delete(ctx context.Context, tokenHash string) error

	RevokeAllForUser(ctx context.Context, userID int64) error
}

type DepartmentRepository interface {
	Create(ctx context.Context, in models.DepartmentInput) (*models.Department, error)
	List(ctx context.Context) ([]models.Department, error)
	GetByID(ctx context.Context, id int64) (*models.Department, error)
	GetByName(ctx context.Context, name string) (*models.Department, error)
	Update(ctx context.Context, id int64, in models.DepartmentInput) (*models.Department, error)
	Delete(ctx context.Context, id int64) error
}

type PositionRepository interface {
	Create(ctx context.Context, in models.PositionInput) (*models.Position, error)
	List(ctx context.Context) ([]models.Position, error)
	GetByID(ctx context.Context, id int64) (*models.Position, error)
	GetByName(ctx context.Context, name string) (*models.Position, error)
	Update(ctx context.Context, id int64, in models.PositionInput) (*models.Position, error)
	Delete(ctx context.Context, id int64) error
}

type EmployeeRepository interface {
	Create(ctx context.Context, in models.EmployeeInput, passwordHash string, roleID int64) (*models.Employee, error)
	GetByID(ctx context.Context, id int64) (*models.Employee, error)

	GetByUserID(ctx context.Context, userID int64) (*models.Employee, error)

	List(ctx context.Context, f models.EmployeeFilter) ([]models.Employee, int64, error)
	Update(ctx context.Context, id int64, in models.EmployeeUpdate) (*models.Employee, error)

	Terminate(ctx context.Context, id int64) error
}

type SalaryRepository interface {
	Create(ctx context.Context, employeeID int64, in models.SalaryInput) (*models.Salary, error)
	ListByEmployee(ctx context.Context, employeeID int64) ([]models.Salary, error)
	DepartmentReport(ctx context.Context) ([]models.DepartmentReportRow, error)
}

type TimeOffRepository interface {
	Create(ctx context.Context, employeeID int64, in models.TimeOffInput) (*models.TimeOff, error)
	List(ctx context.Context, f models.TimeOffFilter) ([]models.TimeOff, error)
	GetByID(ctx context.Context, id int64) (*models.TimeOff, error)
	Decide(ctx context.Context, id int64, status string, approvedBy int64) (*models.TimeOff, error)
}

type HistoryRepository interface {
	ListByEmployee(ctx context.Context, employeeID int64) ([]models.PositionHistory, error)
}

type Repositories struct {
	User       UserRepository
	Role       RoleRepository
	Refresh    RefreshTokenRepository
	Department DepartmentRepository
	Position   PositionRepository
	Employee   EmployeeRepository
	Salary     SalaryRepository
	TimeOff    TimeOffRepository
	History    HistoryRepository
}
