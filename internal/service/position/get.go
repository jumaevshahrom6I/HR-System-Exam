package position

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Get(ctx context.Context, id int64) (*models.Position, error) {
	p, err := s.positions.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get position: %w", err)
	}
	return p, nil
}
