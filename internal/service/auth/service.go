package auth

import (
	"time"

	"hr-system/internal/repository"
	"hr-system/internal/service"
)

type Config struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type Service struct {
	users   repository.UserRepository
	refresh repository.RefreshTokenRepository
	cfg     Config
	now     func() time.Time
}

var _ service.AuthService = (*Service)(nil)

func NewService(users repository.UserRepository, refresh repository.RefreshTokenRepository, cfg Config) *Service {
	return &Service{users: users, refresh: refresh, cfg: cfg, now: time.Now}
}
