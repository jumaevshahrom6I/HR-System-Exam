package employee

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Update(ctx context.Context, id int64, in models.EmployeeUpdate) (*models.Employee, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if in.Status == statusTerminated {
		return nil, fmt.Errorf("%w: use DELETE to terminate an employee", models.ErrInvalidInput)
	}

	current, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}
	if current.Status == statusTerminated {
		return nil, fmt.Errorf("%w: employee is terminated", models.ErrConflict)
	}

	if err := s.checkRefs(ctx, in.DepartmentID, in.PositionID); err != nil {
		return nil, err
	}

	e, err := s.employees.Update(ctx, id, in)
	if err != nil {
		return nil, fmt.Errorf("update employee: %w", err)
	}
	return e, nil
}
