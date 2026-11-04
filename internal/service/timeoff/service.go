package timeoff

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Service struct {
	timeoffs  repository.TimeOffRepository
	employees repository.EmployeeRepository
}

var _ service.TimeOffService = (*Service)(nil)

func NewService(timeoffs repository.TimeOffRepository, employees repository.EmployeeRepository) *Service {
	return &Service{timeoffs: timeoffs, employees: employees}
}

func (s *Service) Create(ctx context.Context, cu models.CurrentUser, in models.TimeOffInput) (*models.TimeOff, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	emp, err := s.employees.GetByUserID(ctx, cu.ID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, fmt.Errorf("%w: only employees can create time off requests", models.ErrForbidden)
		}
		return nil, fmt.Errorf("get employee: %w", err)
	}

	t, err := s.timeoffs.Create(ctx, emp.ID, in)
	if err != nil {
		return nil, fmt.Errorf("create time off request: %w", err)
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, cu models.CurrentUser, f models.TimeOffFilter) ([]models.TimeOff, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}

	if !cu.IsHR() {
		emp, err := s.employees.GetByUserID(ctx, cu.ID)
		if err != nil {
			if errors.Is(err, models.ErrNotFound) {
				return []models.TimeOff{}, nil
			}
			return nil, fmt.Errorf("get employee: %w", err)
		}
		f.EmployeeID = &emp.ID
	}

	items, err := s.timeoffs.List(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list time off requests: %w", err)
	}
	return items, nil
}

func (s *Service) UpdateStatus(ctx context.Context, cu models.CurrentUser, id int64, status string) (*models.TimeOff, error) {
	if !cu.IsHR() {
		return nil, fmt.Errorf("%w: only HR can change request status", models.ErrForbidden)
	}
	if err := models.ValidateTimeOffDecision(status); err != nil {
		return nil, err
	}

	current, err := s.timeoffs.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get time off request: %w", err)
	}
	if current.Status != models.TimeOffPending {
		return nil, fmt.Errorf("%w: request is not pending", models.ErrConflict)
	}

	t, err := s.timeoffs.Decide(ctx, id, status, cu.ID)
	if err != nil {
		return nil, fmt.Errorf("update time off status: %w", err)
	}
	return t, nil
}
