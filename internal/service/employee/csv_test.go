package employee_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"hr-system/internal/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExport(t *testing.T) {
	ctx := context.Background()

	t.Run("file: BOM, header, rows", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.EmployeeFilter) bool {
			return f.Page == 1 && f.Limit == 100 && f.Q == "iv"
		})).Return([]models.Employee{*activeEmployee()}, int64(1), nil).Once()
		var buf bytes.Buffer

		err := e.svc.Export(ctx, models.EmployeeFilter{Q: " iv "}, &buf)

		require.NoError(t, err)
		out := buf.String()
		require.True(t, strings.HasPrefix(out, "\xEF\xBB\xBF"))
		lines := strings.Split(strings.TrimSpace(strings.TrimPrefix(out, "\xEF\xBB\xBF")), "\n")
		require.Equal(t, "id,email,first_name,last_name,phone,department,position,hire_date,status", lines[0])
		require.Equal(t, "5,ivan@company.local,Ivan,Petrov,+992900000000,IT,Dev,2026-01-15,active", lines[1])
		require.Len(t, lines, 2)
	})

	t.Run("several pages are combined into one file", func(t *testing.T) {
		e := newEnv(t)
		page := make([]models.Employee, 100)
		for i := range page {
			page[i] = *activeEmployee()
		}
		e.employees.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.EmployeeFilter) bool { return f.Page == 1 })).Return(page, int64(150), nil).Once()
		e.employees.EXPECT().List(mock.Anything, mock.MatchedBy(func(f models.EmployeeFilter) bool { return f.Page == 2 })).Return(page[:50], int64(150), nil).Once()
		var buf bytes.Buffer

		require.NoError(t, e.svc.Export(ctx, models.EmployeeFilter{}, &buf))

		require.Equal(t, 151, strings.Count(buf.String(), "\n"))
	})

	t.Run("Excel formula protection", func(t *testing.T) {
		e := newEnv(t)
		emp := activeEmployee()
		emp.FirstName = "=cmd|' /C calc'!A0"
		emp.Phone = "+992 90 000-00-00"
		e.employees.EXPECT().List(mock.Anything, mock.Anything).Return([]models.Employee{*emp}, int64(1), nil).Once()
		var buf bytes.Buffer

		require.NoError(t, e.svc.Export(ctx, models.EmployeeFilter{}, &buf))

		require.Contains(t, buf.String(), "'=cmd")
		require.Contains(t, buf.String(), "+992 90 000-00-00")
		require.NotContains(t, buf.String(), ",'+992")
	})

	t.Run("empty result: header only", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().List(mock.Anything, mock.Anything).Return([]models.Employee{}, int64(0), nil).Once()
		var buf bytes.Buffer

		require.NoError(t, e.svc.Export(ctx, models.EmployeeFilter{}, &buf))

		require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	})

	t.Run("unknown status: repository is not called", func(t *testing.T) {
		e := newEnv(t)
		var buf bytes.Buffer

		err := e.svc.Export(ctx, models.EmployeeFilter{Status: "weird"}, &buf)

		require.ErrorIs(t, err, models.ErrInvalidInput)
		require.Zero(t, buf.Len())
	})

	t.Run("repository failure", func(t *testing.T) {
		e := newEnv(t)
		e.employees.EXPECT().List(mock.Anything, mock.Anything).Return(nil, int64(0), errBoom).Once()
		var buf bytes.Buffer

		require.ErrorIs(t, e.svc.Export(ctx, models.EmployeeFilter{}, &buf), errBoom)
	})
}

const importHeader = "email,password,first_name,last_name,phone,department,position,hire_date\n"

func expectImportRefs(e *env) {
	e.departments.EXPECT().GetByName(mock.Anything, "IT").Return(&models.Department{ID: 1, Name: "IT"}, nil).Once()
	e.positions.EXPECT().GetByName(mock.Anything, "Dev").Return(&models.Position{ID: 1, Name: "Dev"}, nil).Once()
	e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Department{ID: 1}, nil)
	e.positions.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Position{ID: 1}, nil)
	e.roles.EXPECT().GetByCode(mock.Anything, "employee").Return(&models.Role{ID: 2}, nil)
}

func TestImport_PartialSuccess(t *testing.T) {
	e := newEnv(t)
	expectImportRefs(e)
	e.employees.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in models.EmployeeInput) bool { return in.Email == "a@company.local" }), mock.Anything, int64(2)).Return(activeEmployee(), nil).Once()
	e.employees.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in models.EmployeeInput) bool { return in.Email == "dup@company.local" }), mock.Anything, int64(2)).Return(nil, models.ErrConflict).Once()
	e.employees.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in models.EmployeeInput) bool { return in.Email == "c@company.local" }), mock.Anything, int64(2)).Return(activeEmployee(), nil).Once()
	file := importHeader +
		"a@company.local,Pass12345,Ann,One,+1,IT,Dev,2026-01-10\n" +
		"dup@company.local,Pass12345,Dup,Two,+2,IT,Dev,2026-01-11\n" +
		"c@company.local,Pass12345,Cid,Three,+3,IT,Dev,2026-01-12\n"

	res, err := e.svc.Import(context.Background(), strings.NewReader(file))

	require.NoError(t, err)
	require.Equal(t, 2, res.Created)
	require.Equal(t, []models.ImportRowError{{Row: 3, Error: "email already registered"}}, res.Errors)
}

func TestImport_RowErrors(t *testing.T) {
	e := newEnv(t)
	e.departments.EXPECT().GetByName(mock.Anything, "IT").Return(&models.Department{ID: 1}, nil).Once()
	e.departments.EXPECT().GetByName(mock.Anything, "Ghost").Return(nil, models.ErrNotFound).Once()
	e.positions.EXPECT().GetByName(mock.Anything, "Dev").Return(&models.Position{ID: 1}, nil).Once()
	file := importHeader +
		"a@company.local,Pass12345,Ann,One,+1,IT,Dev,10.01.2026\n" +
		"b@company.local,Pass12345,Bob,Two,+2,Ghost,Dev,2026-01-11\n" +
		"c@company.local,Pass12345,Cid,Three,,IT,Dev,2026-01-12\n" +
		"d@company.local,short\n"
	e.departments.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Department{ID: 1}, nil).Maybe()
	e.positions.EXPECT().GetByID(mock.Anything, int64(1)).Return(&models.Position{ID: 1}, nil).Maybe()

	res, err := e.svc.Import(context.Background(), strings.NewReader(file))

	require.NoError(t, err)
	require.Zero(t, res.Created)
	require.Equal(t, []models.ImportRowError{
		{Row: 2, Error: "hire_date must be in YYYY-MM-DD format"},
		{Row: 3, Error: "department not found: Ghost"},
		{Row: 4, Error: "phone must be 1-20 characters"},
		{Row: 5, Error: "wrong number of columns"},
	}, res.Errors)
}

func TestImport_FileErrors(t *testing.T) {
	tooMany := importHeader + strings.Repeat("a@company.local,Pass12345,A,B,1,IT,Dev,2026-01-01\n", 1001)
	tests := []struct {
		name string
		file string
	}{
		{"empty file", ""},
		{"no password column", "email,first_name,last_name,phone,department,position,hire_date\n"},
		{"more than 1000 rows", tooMany},
		{"malformed csv", importHeader + "\"unclosed,Pass12345,A,B,1,IT,Dev,2026-01-01\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			_, err := e.svc.Import(context.Background(), strings.NewReader(tt.file))

			require.ErrorIs(t, err, models.ErrInvalidInput)
		})
	}
}

func TestImport_ExcelStyleFile(t *testing.T) {
	e := newEnv(t)
	expectImportRefs(e)
	e.employees.EXPECT().Create(mock.Anything, mock.MatchedBy(func(in models.EmployeeInput) bool {
		return in.Email == "a@company.local" && in.HireDate.Equal(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	}), mock.Anything, int64(2)).Return(activeEmployee(), nil).Once()
	file := "\xEF\xBB\xBFemail;password;first_name;last_name;phone;department;position;hire_date\r\n" +
		"a@company.local;Pass12345;Ann;One;+1;IT;Dev;2026-01-10\r\n"

	res, err := e.svc.Import(context.Background(), strings.NewReader(file))

	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	require.Empty(t, res.Errors)
}
