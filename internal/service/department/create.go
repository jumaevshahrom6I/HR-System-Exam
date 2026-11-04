package department

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Create(ctx context.Context, in models.DepartmentInput) (*models.Department, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	d, err := s.departments.Create(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("create department: %w", err)
	}
	return d, nil
}
