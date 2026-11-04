package history

import (
	"context"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const listSQL = `
SELECT h.id, h.employee_id, h.department_id, d.name AS department_name,
       h.position_id, p.name AS position_name,
       h.start_date, h.end_date, h.created_at
FROM employee_position_history h
JOIN departments d ON d.id = h.department_id
JOIN positions p   ON p.id = h.position_id
WHERE h.employee_id = $1
ORDER BY h.start_date DESC, h.id DESC`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.HistoryRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByEmployee(ctx context.Context, employeeID int64) ([]models.PositionHistory, error) {
	rows, err := r.db.Query(ctx, listSQL, employeeID)
	if err != nil {
		return nil, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	result := make([]models.PositionHistory, 0, len(items))
	for _, m := range items {
		result = append(result, m.toModel())
	}
	return result, nil
}
