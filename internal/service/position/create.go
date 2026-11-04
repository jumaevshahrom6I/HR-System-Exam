package position

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Create(ctx context.Context, in models.PositionInput) (*models.Position, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	p, err := s.positions.Create(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("create position: %w", err)
	}
	return p, nil
}
