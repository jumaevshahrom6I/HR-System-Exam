package service

import (
	"context"
	"io"

	"hr-system/internal/models"
)

type AuthService interface {
	Login(ctx context.Context, in models.LoginInput) (*models.TokenPair, error)

	Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error)

	Logout(ctx context.Context, refreshToken string) error

	Authenticate(accessToken string) (*models.AccessClaims, error)
}

type DepartmentService interface {
	Create(ctx context.Context, in models.DepartmentInput) (*models.Department, error)
	List(ctx context.Context) ([]models.Department, error)
	Get(ctx context.Context, id int64) (*models.Department, error)
	Update(ctx context.Context, id int64, in models.DepartmentInput) (*models.Department, error)
	Delete(ctx context.Context, id int64) error
}

type PositionService interface {
	Create(ctx context.Context, in models.PositionInput) (*models.Position, error)
	List(ctx context.Context) ([]models.Position, error)
	Get(ctx context.Context, id int64) (*models.Position, error)
	Update(ctx context.Context, id int64, in models.PositionInput) (*models.Position, error)
	Delete(ctx context.Context, id int64) error
}

type EmployeeService interface {
	Create(ctx context.Context, in models.EmployeeInput) (*models.Employee, error)
	Get(ctx context.Context, id int64) (*models.Employee, error)
	List(ctx context.Context, f models.EmployeeFilter) (*models.EmployeePage, error)
	Update(ctx context.Context, id int64, in models.EmployeeUpdate) (*models.Employee, error)

	Delete(ctx context.Context, id int64) error

	Me(ctx context.Context, cu models.CurrentUser) (*models.Profile, error)
	History(ctx context.Context, employeeID int64) ([]models.PositionHistory, error)
	Export(ctx context.Context, f models.EmployeeFilter, w io.Writer) error
	Import(ctx context.Context, r io.Reader) (*models.ImportResult, error)
}

type TimeOffService interface {
	Create(ctx context.Context, cu models.CurrentUser, in models.TimeOffInput) (*models.TimeOff, error)
	List(ctx context.Context, cu models.CurrentUser, f models.TimeOffFilter) ([]models.TimeOff, error)
	UpdateStatus(ctx context.Context, cu models.CurrentUser, id int64, status string) (*models.TimeOff, error)
}

type SalaryService interface {
	Add(ctx context.Context, employeeID int64, in models.SalaryInput) (*models.Salary, error)

	ListByEmployee(ctx context.Context, employeeID int64) ([]models.Salary, error)

	ListMine(ctx context.Context, cu models.CurrentUser) ([]models.Salary, error)
}

type ReportService interface {
	DepartmentReport(ctx context.Context) ([]models.DepartmentReportRow, error)
}

type Services struct {
	Auth       AuthService
	Department DepartmentService
	Position   PositionService
	Employee   EmployeeService
	Salary     SalaryService
	Report     ReportService
	TimeOff    TimeOffService
}
