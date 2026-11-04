package department

import (
	"context"
	"fmt"
)

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.departments.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	return nil
}
