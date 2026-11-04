package auth

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"
)

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("%w: refresh_token is required", models.ErrInvalidInput)
	}

	userID, err := s.refresh.Consume(ctx, hashToken(refreshToken))
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, models.ErrInvalidToken
		}
		return nil, fmt.Errorf("consume refresh token: %w", err)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, models.ErrInvalidToken
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	if !user.IsActive {
		return nil, models.ErrInvalidToken
	}

	return s.issueTokens(ctx, user)
}
