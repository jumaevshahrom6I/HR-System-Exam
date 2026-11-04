package timeoff

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID         int64     `db:"id"`
	EmployeeID int64     `db:"employee_id"`
	ApprovedBy *int64    `db:"approved_by"`
	LeaveType  string    `db:"leave_type"`
	StartDate  time.Time `db:"start_date"`
	EndDate    time.Time `db:"end_date"`
	Status     string    `db:"status"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (m dbModel) toModel() models.TimeOff {
	return models.TimeOff{
		ID:         m.ID,
		EmployeeID: m.EmployeeID,
		ApprovedBy: m.ApprovedBy,
		LeaveType:  m.LeaveType,
		StartDate:  m.StartDate,
		EndDate:    m.EndDate,
		Status:     m.Status,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
