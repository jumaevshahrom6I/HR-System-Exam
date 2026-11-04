package history

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID             int64      `db:"id"`
	EmployeeID     int64      `db:"employee_id"`
	DepartmentID   int64      `db:"department_id"`
	DepartmentName string     `db:"department_name"`
	PositionID     int64      `db:"position_id"`
	PositionName   string     `db:"position_name"`
	StartDate      time.Time  `db:"start_date"`
	EndDate        *time.Time `db:"end_date"`
	CreatedAt      time.Time  `db:"created_at"`
}

func (m dbModel) toModel() models.PositionHistory {
	return models.PositionHistory{
		ID:             m.ID,
		EmployeeID:     m.EmployeeID,
		DepartmentID:   m.DepartmentID,
		DepartmentName: m.DepartmentName,
		PositionID:     m.PositionID,
		PositionName:   m.PositionName,
		StartDate:      m.StartDate,
		EndDate:        m.EndDate,
		CreatedAt:      m.CreatedAt,
	}
}
