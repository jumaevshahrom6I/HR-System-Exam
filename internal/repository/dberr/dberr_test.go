package dberr_test

import (
	"errors"
	"fmt"
	"testing"

	"hr-system/internal/models"
	"hr-system/internal/repository/dberr"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name    string
		in      error
		wantIs  error
		wantNil bool
		same    bool
	}{
		{name: "nil stays nil", in: nil, wantNil: true},
		{name: "no rows: not found", in: pgx.ErrNoRows, wantIs: models.ErrNotFound},
		{name: "no rows inside a wrapper", in: fmt.Errorf("query: %w", pgx.ErrNoRows), wantIs: models.ErrNotFound},
		{name: "duplicate 23505: conflict", in: &pgconn.PgError{Code: "23505"}, wantIs: models.ErrConflict},
		{name: "foreign key 23503: conflict", in: &pgconn.PgError{Code: "23503"}, wantIs: models.ErrConflict},
		{name: "other Postgres error is returned as is", in: &pgconn.PgError{Code: "42P01"}, same: true},
		{name: "plain error is returned as is", in: boom, same: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dberr.Map(tt.in)

			switch {
			case tt.wantNil:
				require.NoError(t, got)
			case tt.same:
				require.Equal(t, tt.in, got)
				require.NotErrorIs(t, got, models.ErrNotFound)
				require.NotErrorIs(t, got, models.ErrConflict)
			default:
				require.ErrorIs(t, got, tt.wantIs)
			}
		})
	}
}
