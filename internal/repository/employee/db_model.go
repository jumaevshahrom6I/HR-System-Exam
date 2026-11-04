package employee

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID              int64      `db:"id"`
	UserID          int64      `db:"user_id"`
	Email           string     `db:"email"`
	FirstName       string     `db:"first_name"`
	LastName        string     `db:"last_name"`
	Phone           string     `db:"phone"`
	DepartmentID    int64      `db:"department_id"`
	DepartmentName  string     `db:"department_name"`
	PositionID      int64      `db:"position_id"`
	PositionName    string     `db:"position_name"`
	HireDate        time.Time  `db:"hire_date"`
	TerminationDate *time.Time `db:"termination_date"`
	Status          string     `db:"status"`
}

func (m dbModel) toModel() models.Employee {
	return models.Employee{
		ID:              m.ID,
		UserID:          m.UserID,
		Email:           m.Email,
		FirstName:       m.FirstName,
		LastName:        m.LastName,
		Phone:           m.Phone,
		DepartmentID:    m.DepartmentID,
		DepartmentName:  m.DepartmentName,
		PositionID:      m.PositionID,
		PositionName:    m.PositionName,
		HireDate:        m.HireDate,
		TerminationDate: m.TerminationDate,
		Status:          m.Status,
	}
}
