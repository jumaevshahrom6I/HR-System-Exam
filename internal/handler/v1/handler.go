package v1

import (
	"context"

	"hr-system/internal/service"
)

type Check struct {
	Name string
	Ping func(ctx context.Context) error
}

type Handler struct {
	services *service.Services
	checks   []Check
}

func NewHandler(services *service.Services, checks []Check) *Handler {
	return &Handler{services: services, checks: checks}
}
