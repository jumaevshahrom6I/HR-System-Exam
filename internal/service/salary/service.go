package salary

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Service struct {
	salaries  repository.SalaryRepository
	employees repository.EmployeeRepository
}

var _ service.SalaryService = (*Service)(nil)

func NewService(salaries repository.SalaryRepository, employees repository.EmployeeRepository) *Service {
	return &Service{salaries: salaries, employees: employees}
}

func (s *Service) Add(ctx context.Context, employeeID int64, in models.SalaryInput) (*models.Salary, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}
	sal, err := s.salaries.Create(ctx, employeeID, in)
	if err != nil {
		return nil, fmt.Errorf("create salary: %w", err)
	}
	return sal, nil
}

func (s *Service) ListByEmployee(ctx context.Context, employeeID int64) ([]models.Salary, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}
	items, err := s.salaries.ListByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list salaries: %w", err)
	}
	return items, nil
}

func (s *Service) ListMine(ctx context.Context, cu models.CurrentUser) ([]models.Salary, error) {
	emp, err := s.employees.GetByUserID(ctx, cu.ID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return []models.Salary{}, nil
		}
		return nil, fmt.Errorf("get employee: %w", err)
	}
	items, err := s.salaries.ListByEmployee(ctx, emp.ID)
	if err != nil {
		return nil, fmt.Errorf("list salaries: %w", err)
	}
	return items, nil
}
