package models

import (
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID           int64
	RoleID       int64
	RoleCode     string
	Email        string
	PasswordHash string
	IsActive     bool
}

type LoginInput struct {
	Email    string
	Password string
}

func (in *LoginInput) Validate() error {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if in.Password == "" {
		return fmt.Errorf("%w: password is required", ErrInvalidInput)
	}
	return nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessClaims struct {
	UserID int64
	Role   string
}

type RefreshToken struct {
	UserID    int64
	ExpiresAt time.Time
}

type CurrentUser struct {
	ID   int64
	Role string
}

func (u CurrentUser) IsHR() bool { return u.Role == RoleHR }
