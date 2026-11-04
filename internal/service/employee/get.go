package employee

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Get(ctx context.Context, id int64) (*models.Employee, error) {
	e, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}
	return e, nil
}
