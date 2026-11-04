package position

import (
	"context"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const columns = `id, name, base_salary::float8 AS base_salary, created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.PositionRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, in models.PositionInput) (*models.Position, error) {
	return r.getOne(ctx, `INSERT INTO positions (name, base_salary) VALUES ($1, $2::float8) RETURNING `+columns, in.Name, in.BaseSalary)
}

func (r *Repository) List(ctx context.Context) ([]models.Position, error) {
	rows, err := r.db.Query(ctx, `SELECT `+columns+` FROM positions ORDER BY id`)
	if err != nil {
		return nil, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	result := make([]models.Position, 0, len(items))
	for _, m := range items {
		result = append(result, m.toModel())
	}
	return result, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*models.Position, error) {
	return r.getOne(ctx, `SELECT `+columns+` FROM positions WHERE id = $1`, id)
}

func (r *Repository) GetByName(ctx context.Context, name string) (*models.Position, error) {
	return r.getOne(ctx, `SELECT `+columns+` FROM positions WHERE name = $1`, name)
}

func (r *Repository) Update(ctx context.Context, id int64, in models.PositionInput) (*models.Position, error) {
	return r.getOne(ctx, `UPDATE positions SET name = $2, base_salary = $3::float8, updated_at = now() WHERE id = $1 RETURNING `+columns, id, in.Name, in.BaseSalary)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM positions WHERE id = $1`, id)
	if err != nil {
		return dberr.Map(err)
	}
	if tag.RowsAffected() == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repository) getOne(ctx context.Context, sql string, args ...any) (*models.Position, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	p := m.toModel()
	return &p, nil
}
