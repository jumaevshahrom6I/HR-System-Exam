package role

import "hr-system/internal/models"

type dbModel struct {
	ID   int64  `db:"id"`
	Code string `db:"code"`
	Name string `db:"name"`
}

func (m dbModel) toModel() models.Role {
	return models.Role{ID: m.ID, Code: m.Code, Name: m.Name}
}
