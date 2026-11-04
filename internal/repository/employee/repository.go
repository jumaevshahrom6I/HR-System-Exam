package employee

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hr-system/internal/models"
	"hr-system/internal/repository"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const selectEmployee = `
SELECT e.id, e.user_id, u.email, e.first_name, e.last_name, e.phone,
       e.department_id, d.name AS department_name,
       e.position_id, p.name AS position_name,
       e.hire_date, e.termination_date, e.status
FROM employees e
JOIN users u       ON u.id = e.user_id
JOIN departments d ON d.id = e.department_id
JOIN positions p   ON p.id = e.position_id`

type Repository struct {
	db *pgxpool.Pool
}

var _ repository.EmployeeRepository = (*Repository)(nil)

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, in models.EmployeeInput, passwordHash string, roleID int64) (*models.Employee, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO users (role_id, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		roleID, in.Email, passwordHash,
	).Scan(&userID)
	if err != nil {
		return nil, dberr.Map(err)
	}

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO employees (user_id, department_id, position_id, first_name, last_name, phone, hire_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		userID, in.DepartmentID, in.PositionID, in.FirstName, in.LastName, in.Phone, in.HireDate,
	).Scan(&id)
	if err != nil {
		return nil, dberr.Map(err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO employee_position_history (employee_id, department_id, position_id, start_date)
		 VALUES ($1, $2, $3, $4)`,
		id, in.DepartmentID, in.PositionID, in.HireDate,
	)
	if err != nil {
		return nil, dberr.Map(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	return r.getOne(ctx, selectEmployee+` WHERE e.id = $1`, id)
}

func (r *Repository) GetByUserID(ctx context.Context, userID int64) (*models.Employee, error) {
	return r.getOne(ctx, selectEmployee+` WHERE e.user_id = $1`, userID)
}

func (r *Repository) List(ctx context.Context, f models.EmployeeFilter) ([]models.Employee, int64, error) {
	var (
		conds []string
		args  []any
	)

	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.Q != "" {
		add(`(e.first_name ILIKE $%[1]d OR e.last_name ILIKE $%[1]d OR u.email ILIKE $%[1]d)`, "%"+escapeLike(f.Q)+"%")
	}
	if f.DepartmentID != nil {
		add(`e.department_id = $%d`, *f.DepartmentID)
	}
	if f.PositionID != nil {
		add(`e.position_id = $%d`, *f.PositionID)
	}
	if f.Status != "" {
		add(`e.status = $%d`, f.Status)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	var total int64
	countSQL := `SELECT count(*) FROM employees e JOIN users u ON u.id = e.user_id` + where
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, dberr.Map(err)
	}

	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	pageSQL := selectEmployee + where +
		fmt.Sprintf(` ORDER BY e.last_name, e.first_name, e.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args))
	rows, err := r.db.Query(ctx, pageSQL, args...)
	if err != nil {
		return nil, 0, dberr.Map(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, 0, dberr.Map(err)
	}

	result := make([]models.Employee, 0, len(items))
	for _, m := range items {
		result = append(result, m.toModel())
	}
	return result, total, nil
}

func (r *Repository) Update(ctx context.Context, id int64, in models.EmployeeUpdate) (*models.Employee, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		currentDepartment int64
		currentPosition   int64
		hireDate          time.Time
	)
	err = tx.QueryRow(ctx,
		`SELECT department_id, position_id, hire_date FROM employees WHERE id = $1 FOR UPDATE`, id,
	).Scan(&currentDepartment, &currentPosition, &hireDate)
	if err != nil {
		return nil, dberr.Map(err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE employees
		 SET department_id = $2, position_id = $3, first_name = $4, last_name = $5, phone = $6, status = $7, updated_at = now()
		 WHERE id = $1`,
		id, in.DepartmentID, in.PositionID, in.FirstName, in.LastName, in.Phone, in.Status,
	)
	if err != nil {
		return nil, dberr.Map(err)
	}

	if currentDepartment != in.DepartmentID || currentPosition != in.PositionID {
		_, err = tx.Exec(ctx,
			`UPDATE employee_position_history
			 SET end_date = GREATEST(CURRENT_DATE, $2::date)
			 WHERE employee_id = $1 AND end_date IS NULL`,
			id, hireDate,
		)
		if err != nil {
			return nil, dberr.Map(err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO employee_position_history (employee_id, department_id, position_id, start_date)
			 VALUES ($1, $2, $3, GREATEST(CURRENT_DATE, $4::date))`,
			id, in.DepartmentID, in.PositionID, hireDate,
		)
		if err != nil {
			return nil, dberr.Map(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Terminate(ctx context.Context, id int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(ctx,
		`UPDATE employees
		 SET status = 'terminated', termination_date = GREATEST(CURRENT_DATE, hire_date), updated_at = now()
		 WHERE id = $1
		 RETURNING user_id`, id,
	).Scan(&userID)
	if err != nil {
		return dberr.Map(err)
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET is_active = false, updated_at = now() WHERE id = $1`, userID); err != nil {
		return dberr.Map(err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE employee_position_history
		 SET end_date = GREATEST(CURRENT_DATE, start_date)
		 WHERE employee_id = $1 AND end_date IS NULL`, id,
	)
	if err != nil {
		return dberr.Map(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *Repository) getOne(ctx context.Context, sql string, args ...any) (*models.Employee, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, dberr.Map(err)
	}
	m, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[dbModel])
	if err != nil {
		return nil, dberr.Map(err)
	}
	e := m.toModel()
	return &e, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string { return likeEscaper.Replace(s) }
