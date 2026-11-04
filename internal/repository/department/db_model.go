package department

import (
	"time"

	"hr-system/internal/models"
)

type dbModel struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (m dbModel) toModel() models.Department {
	return models.Department{ID: m.ID, Name: m.Name, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
