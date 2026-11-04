package position

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) List(ctx context.Context) ([]models.Position, error) {
	items, err := s.positions.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	return items, nil
}
