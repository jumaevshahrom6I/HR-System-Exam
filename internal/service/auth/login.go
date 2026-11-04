package auth

import (
	"context"
	"errors"
	"fmt"

	"hr-system/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Login(ctx context.Context, in models.LoginInput) (*models.TokenPair, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, models.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if !user.IsActive {
		return nil, models.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, models.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}
