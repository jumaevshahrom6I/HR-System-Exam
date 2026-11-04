package timeoff

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const columns = `id, employee_id, approved_by, leave_type, start_date, end_date, status, created_at, updated_at`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.TimeOffRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, employeeID int64, in models.TimeOffInput) (*models.TimeOff, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, employeeID); err != nil {
		return nil, dberr.Map(err)
	}

	rows, err := tx.Query(ctx,
		`INSERT INTO time_off (employee_id, leave_type, start_date, end_date)
		 SELECT $1::int, $2::text, $3::date, $4::date
		 WHERE NOT EXISTS (
		     SELECT 1 FROM time_off
		     WHERE employee_id = $1::int
		       AND status IN ('pending', 'approved')
		       AND start_date <= $4::date AND end_date >= $3::date
		 )
		 RETURNING `+columns,
		employeeID, in.LeaveType, in.StartDate, in.EndDate,
	)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: dates overlap with another request", models.ErrConflict)
	}
	if err != nil {
		return nil, dberr.Map(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	t := m.toModel()
	return &t, nil
}

func (r *Repository) List(ctx context.Context, f models.TimeOffFilter) ([]models.TimeOff, error) {
	var (
		conds []string
		args  []any
	)
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.EmployeeID != nil {
		args = append(args, *f.EmployeeID)
		conds = append(conds, fmt.Sprintf("employee_id = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	rows, err := r.db.Query(ctx, `SELECT `+columns+` FROM time_off`+where+` ORDER BY id DESC`, args...)
	if err != nil {
		return nil, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	result := make([]models.TimeOff, 0, len(items))
	for _, m := range items {
		result = append(result, m.toModel())
	}
	return result, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*models.TimeOff, error) {
	rows, err := r.db.Query(ctx, `SELECT `+columns+` FROM time_off WHERE id = $1`, id)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	t := m.toModel()
	return &t, nil
}

func (r *Repository) Decide(ctx context.Context, id int64, status string, approvedBy int64) (*models.TimeOff, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE time_off SET status = $2, approved_by = $3, updated_at = now()
		 WHERE id = $1 AND status = 'pending'
		 RETURNING `+columns,
		id, status, approvedBy,
	)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: request is not pending", models.ErrConflict)
	}
	if err != nil {
		return nil, dberr.Map(err)
	}
	t := m.toModel()
	return &t, nil
}
