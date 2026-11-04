package department

import (
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Service struct {
	departments repository.DepartmentRepository
}

var _ service.DepartmentService = (*Service)(nil)

func NewService(departments repository.DepartmentRepository) *Service {
	return &Service{departments: departments}
}
