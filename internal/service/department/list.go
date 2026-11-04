package department

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) List(ctx context.Context) ([]models.Department, error) {
	items, err := s.departments.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	return items, nil
}
