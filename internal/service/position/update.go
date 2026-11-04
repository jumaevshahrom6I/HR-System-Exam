package position

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Update(ctx context.Context, id int64, in models.PositionInput) (*models.Position, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	p, err := s.positions.Update(ctx, id, in)
	if err != nil {
		return nil, fmt.Errorf("update position: %w", err)
	}
	return p, nil
}
