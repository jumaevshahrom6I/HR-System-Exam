package employee

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

const (
	roleEmployee     = "employee"
	statusTerminated = "terminated"
)

type Service struct {
	employees   repository.EmployeeRepository
	users       repository.UserRepository
	roles       repository.RoleRepository
	departments repository.DepartmentRepository
	positions   repository.PositionRepository
	refresh     repository.RefreshTokenRepository
	history     repository.HistoryRepository
}

var _ service.EmployeeService = (*Service)(nil)

func NewService(
	employees repository.EmployeeRepository,
	users repository.UserRepository,
	roles repository.RoleRepository,
	departments repository.DepartmentRepository,
	positions repository.PositionRepository,
	refresh repository.RefreshTokenRepository,
	history repository.HistoryRepository,
) *Service {
	return &Service{
		employees:   employees,
		users:       users,
		roles:       roles,
		departments: departments,
		positions:   positions,
		refresh:     refresh,
		history:     history,
	}
}

func (s *Service) checkRefs(ctx context.Context, departmentID, positionID int64) error {
	if _, err := s.departments.GetByID(ctx, departmentID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("%w: department does not exist", models.ErrInvalidInput)
		}
		return fmt.Errorf("get department: %w", err)
	}
	if _, err := s.positions.GetByID(ctx, positionID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return fmt.Errorf("%w: position does not exist", models.ErrInvalidInput)
		}
		return fmt.Errorf("get position: %w", err)
	}
	return nil
}
