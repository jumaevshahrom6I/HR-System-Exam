package user

import (
	"context"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const selectUser = `
SELECT u.id, u.role_id, r.code AS role_code, u.email, u.password_hash, u.is_active
FROM users u
JOIN roles r ON r.id = u.role_id`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.UserRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.getOne(ctx, selectUser+` WHERE u.email = $1`, email)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return r.getOne(ctx, selectUser+` WHERE u.id = $1`, id)
}

func (r *Repository) getOne(ctx context.Context, sql string, args ...any) (*models.User, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, dberr.Map(err)
	}

	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	u := m.toModel()
	return &u, nil
}
