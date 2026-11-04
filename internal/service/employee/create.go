package employee

import (
	"context"
	"fmt"

	"hr-system/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Create(ctx context.Context, in models.EmployeeInput) (*models.Employee, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	if err := s.checkRefs(ctx, in.DepartmentID, in.PositionID); err != nil {
		return nil, err
	}

	role, err := s.roles.GetByCode(ctx, roleEmployee)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	e, err := s.employees.Create(ctx, in, string(hash), role.ID)
	if err != nil {
		return nil, fmt.Errorf("create employee: %w", err)
	}
	return e, nil
}
