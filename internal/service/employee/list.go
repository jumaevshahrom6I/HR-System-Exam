package employee

import (
	"context"
	"fmt"
	"strings"

	"hr-system/internal/models"
)

const maxLimit = 100

func (s *Service) List(ctx context.Context, f models.EmployeeFilter) (*models.EmployeePage, error) {
	f.Q = strings.TrimSpace(f.Q)
	if f.Page < 1 {
		return nil, fmt.Errorf("%w: page must be >= 1", models.ErrInvalidInput)
	}
	if f.Limit < 1 || f.Limit > maxLimit {
		return nil, fmt.Errorf("%w: limit must be 1-%d", models.ErrInvalidInput, maxLimit)
	}
	if err := checkStatus(f.Status); err != nil {
		return nil, err
	}

	items, total, err := s.employees.List(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	return &models.EmployeePage{Items: items, Total: total, Page: f.Page, Limit: f.Limit}, nil
}

func checkStatus(status string) error {
	switch status {
	case "", "active", "on_leave", statusTerminated:
		return nil
	}
	return fmt.Errorf("%w: status must be active, on_leave or terminated", models.ErrInvalidInput)
}
