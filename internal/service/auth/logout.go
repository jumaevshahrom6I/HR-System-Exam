package auth

import (
	"context"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return fmt.Errorf("%w: refresh_token is required", models.ErrInvalidInput)
	}
	if err := s.refresh.Delete(ctx, hashToken(refreshToken)); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
