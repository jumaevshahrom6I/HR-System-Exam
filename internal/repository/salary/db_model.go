package salary

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID            int64     `db:"id"`
	EmployeeID    int64     `db:"employee_id"`
	Amount        float64   `db:"amount"`
	EffectiveDate time.Time `db:"effective_date"`
	CreatedAt     time.Time `db:"created_at"`
}

func (m dbModel) toModel() models.Salary {
	return models.Salary{ID: m.ID, EmployeeID: m.EmployeeID, Amount: m.Amount, EffectiveDate: m.EffectiveDate, CreatedAt: m.CreatedAt}
}

type reportModel struct {
	DepartmentID   int64   `db:"department_id"`
	Department     string  `db:"department"`
	EmployeesCount int64   `db:"employees_count"`
	TotalSalary    float64 `db:"total_salary"`
	AvgSalary      float64 `db:"avg_salary"`
}
