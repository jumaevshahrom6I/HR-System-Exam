package position

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID         int64     `db:"id"`
	Name       string    `db:"name"`
	BaseSalary float64   `db:"base_salary"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (m dbModel) toModel() models.Position {
	return models.Position{ID: m.ID, Name: m.Name, BaseSalary: m.BaseSalary, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
