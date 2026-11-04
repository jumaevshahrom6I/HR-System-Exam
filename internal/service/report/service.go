package report

import (
	"context"
	"fmt"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Service struct {
	salaries repository.SalaryRepository
}

var _ service.ReportService = (*Service)(nil)

func NewService(salaries repository.SalaryRepository) *Service {
	return &Service{salaries: salaries}
}

func (s *Service) DepartmentReport(ctx context.Context) ([]models.DepartmentReportRow, error) {
	rows, err := s.salaries.DepartmentReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("department report: %w", err)
	}
	return rows, nil
}
