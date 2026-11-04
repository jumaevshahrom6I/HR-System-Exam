package department

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Get(ctx context.Context, id int64) (*models.Department, error) {
	d, err := s.departments.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get department: %w", err)
	}
	return d, nil
}
