package position

import (
	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Service struct {
	positions repository.PositionRepository
}

var _ service.PositionService = (*Service)(nil)

func NewService(positions repository.PositionRepository) *Service {
	return &Service{positions: positions}
}
