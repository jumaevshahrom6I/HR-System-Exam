package employee

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Delete(ctx context.Context, id int64) error {
	e, err := s.employees.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get employee: %w", err)
	}
	if e.Status == statusTerminated {
		return fmt.Errorf("%w: employee is already terminated", models.ErrConflict)
	}

	if err := s.employees.Terminate(ctx, id); err != nil {
		return fmt.Errorf("terminate employee: %w", err)
	}

	if err := s.refresh.RevokeAllForUser(ctx, e.UserID); err != nil {
		return fmt.Errorf("revoke refresh tokens: %w", err)
	}
	return nil
}
