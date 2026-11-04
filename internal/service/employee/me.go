package employee

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Me(ctx context.Context, cu models.CurrentUser) (*models.Profile, error) {
	user, err := s.users.GetByID(ctx, cu.ID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var emp *models.Employee
	e, err := s.employees.GetByUserID(ctx, cu.ID)
	switch {
	case err == nil:
		emp = e
	case errors.Is(err, models.ErrNotFound):

	default:
		return nil, fmt.Errorf("get employee: %w", err)
	}

	return &models.Profile{UserID: user.ID, Email: user.Email, Role: user.RoleCode, Employee: emp}, nil
}
