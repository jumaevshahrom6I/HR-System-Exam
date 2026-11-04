package position

import (
	"context"
	"fmt"
)

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.positions.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete position: %w", err)
	}
	return nil
}
