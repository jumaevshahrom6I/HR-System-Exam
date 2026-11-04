package department

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Update(ctx context.Context, id int64, in models.DepartmentInput) (*models.Department, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	d, err := s.departments.Update(ctx, id, in)
	if err != nil {
		return nil, fmt.Errorf("update department: %w", err)
	}
	return d, nil
}
