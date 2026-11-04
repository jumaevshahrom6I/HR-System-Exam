package employee

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) History(ctx context.Context, employeeID int64) ([]models.PositionHistory, error) {
	if _, err := s.employees.GetByID(ctx, employeeID); err != nil {
		return nil, fmt.Errorf("get employee: %w", err)
	}
	items, err := s.history.ListByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list position history: %w", err)
	}
	return items, nil
}
