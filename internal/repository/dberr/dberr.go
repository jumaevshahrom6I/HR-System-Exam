package dberr

import (
	"errors"
	"fmt"

	"hr-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
)

func Map(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return models.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case codeUniqueViolation:
			return fmt.Errorf("%w: already exists", models.ErrConflict)
		case codeForeignKeyViolation:
			return fmt.Errorf("%w: related records conflict", models.ErrConflict)
		}
	}

	return err
}
