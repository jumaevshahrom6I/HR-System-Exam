package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

func TestLogin(t *testing.T) {
	tests := []struct {
		name    string
		in      models.LoginInput
		setup   func(e *env)
		wantErr error
	}{
		{
			name: "success: both tokens issued, refresh saved once",
			in:   models.LoginInput{Email: "hr@company.local", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
				e.refresh.EXPECT().Create(mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("models.RefreshToken")).Return(nil).Once()
			},
		},
		{
			name: "upper-case email is looked up in lower case",
			in:   models.LoginInput{Email: "  HR@Company.Local ", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
				e.refresh.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "unknown email",
			in:   models.LoginInput{Email: "nobody@company.local", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "nobody@company.local").Return(nil, models.ErrNotFound).Once()
			},
			wantErr: models.ErrInvalidCredentials,
		},
		{
			name: "wrong password",
			in:   models.LoginInput{Email: "hr@company.local", Password: "wrong-password"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
			},
			wantErr: models.ErrInvalidCredentials,
		},
		{
			name: "account is inactive",
			in:   models.LoginInput{Email: "off@company.local", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "off@company.local").Return(newUser(t, 2, "off@company.local", "Pass12345", "employee", false), nil).Once()
			},
			wantErr: models.ErrInvalidCredentials,
		},
		{
			name:    "empty email: repository is not called",
			in:      models.LoginInput{Email: "", Password: "Pass12345"},
			setup:   func(*env) {},
			wantErr: models.ErrInvalidInput,
		},
		{
			name:    "empty password: repository is not called",
			in:      models.LoginInput{Email: "hr@company.local", Password: ""},
			setup:   func(*env) {},
			wantErr: models.ErrInvalidInput,
		},
		{
			name: "repository failure is returned as an error, not as invalid credentials",
			in:   models.LoginInput{Email: "hr@company.local", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(nil, errBoom).Once()
			},
			wantErr: errBoom,
		},
		{
			name: "refresh token save failure",
			in:   models.LoginInput{Email: "hr@company.local", Password: "Pass12345"},
			setup: func(e *env) {
				e.users.EXPECT().GetByEmail(mock.Anything, "hr@company.local").Return(newUser(t, 1, "hr@company.local", "Pass12345", "hr", true), nil).Once()
				e.refresh.EXPECT().Create(mock.Anything, mock.Anything, mock.Anything).Return(errBoom).Once()
			},
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, 15*time.Minute)
			tt.setup(e)

			pair, err := e.svc.Login(context.Background(), tt.in)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.wantErr == errBoom {
					require.NotErrorIs(t, err, models.ErrInvalidCredentials)
				}
				require.Nil(t, pair)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, pair.AccessToken)
			require.NotEmpty(t, pair.RefreshToken)
		})
	}
}
