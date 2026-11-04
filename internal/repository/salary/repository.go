package salary

import (
	"context"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const columns = `id, employee_id, amount::float8 AS amount, effective_date, created_at`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.SalaryRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, employeeID int64, in models.SalaryInput) (*models.Salary, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO salaries (employee_id, amount, effective_date) VALUES ($1, $2::float8, $3) RETURNING `+columns,
		employeeID, in.Amount, in.EffectiveDate)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	s := m.toModel()
	return &s, nil
}

func (r *Repository) ListByEmployee(ctx context.Context, employeeID int64) ([]models.Salary, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+columns+` FROM salaries WHERE employee_id = $1 ORDER BY effective_date DESC, id DESC`, employeeID)
	if err != nil {
		return nil, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	result := make([]models.Salary, 0, len(items))
	for _, m := range items {
		result = append(result, m.toModel())
	}
	return result, nil
}

const reportSQL = `
SELECT d.id AS department_id, d.name AS department,
       COUNT(e.id) AS employees_count,
       COALESCE(SUM(cs.amount), 0)::float8 AS total_salary,
       COALESCE(AVG(cs.amount), 0)::float8 AS avg_salary
FROM departments d
LEFT JOIN employees e ON e.department_id = d.id AND e.status <> 'terminated'
LEFT JOIN LATERAL (
    SELECT s.amount FROM salaries s
    WHERE s.employee_id = e.id AND s.effective_date <= CURRENT_DATE
    ORDER BY s.effective_date DESC LIMIT 1
) cs ON true
GROUP BY d.id, d.name
ORDER BY d.name`

func (r *Repository) DepartmentReport(ctx context.Context) ([]models.DepartmentReportRow, error) {
	rows, err := r.db.Query(ctx, reportSQL)
	if err != nil {
		return nil, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[reportModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	result := make([]models.DepartmentReportRow, 0, len(items))
	for _, m := range items {
		result = append(result, models.DepartmentReportRow{
			DepartmentID:   m.DepartmentID,
			Department:     m.Department,
			EmployeesCount: m.EmployeesCount,
			TotalSalary:    m.TotalSalary,
			AvgSalary:      m.AvgSalary,
		})
	}
	return result, nil
}
