package user

import "hr-system/internal/models"

type dbModel struct {
	ID           int64  `db:"id"`
	RoleID       int64  `db:"role_id"`
	RoleCode     string `db:"role_code"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
	IsActive     bool   `db:"is_active"`
}

func (m dbModel) toModel() models.User {
	return models.User{
		ID:           m.ID,
		RoleID:       m.RoleID,
		RoleCode:     m.RoleCode,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		IsActive:     m.IsActive,
	}
}
