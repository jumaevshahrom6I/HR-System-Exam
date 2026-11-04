package role

import (
	"context"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.RoleRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*models.Role, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, name FROM roles WHERE code = $1`, code)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	role := m.toModel()
	return &role, nil
}
