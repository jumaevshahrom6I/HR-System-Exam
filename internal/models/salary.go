package models

import (
	"fmt"
	"time"
)

type Salary struct {
	ID            int64
	EmployeeID    int64
	Amount        float64
	EffectiveDate time.Time
	CreatedAt     time.Time
}

type SalaryInput struct {
	Amount        float64
	EffectiveDate time.Time
}

func (in *SalaryInput) Validate() error {
	if in.Amount <= 0 {
		return fmt.Errorf("%w: amount must be > 0", ErrInvalidInput)
	}
	if in.EffectiveDate.IsZero() {
		return fmt.Errorf("%w: effective_date is required", ErrInvalidInput)
	}
	return nil
}

type DepartmentReportRow struct {
	DepartmentID   int64
	Department     string
	EmployeesCount int64
	TotalSalary    float64
	AvgSalary      float64
}
