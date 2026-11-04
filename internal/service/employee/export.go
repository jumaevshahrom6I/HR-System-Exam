package employee

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"hr-system/internal/models"
)

const exportPageSize = 100

var exportHeader = []string{"id", "email", "first_name", "last_name", "phone", "department", "position", "hire_date", "status"}

func (s *Service) Export(ctx context.Context, f models.EmployeeFilter, w io.Writer) error {
	f.Q = strings.TrimSpace(f.Q)
	if err := checkStatus(f.Status); err != nil {
		return err
	}

	if _, err := io.WriteString(w, "\xEF\xBB\xBF"); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(exportHeader); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	f.Page, f.Limit = 1, exportPageSize
	for {
		items, total, err := s.employees.List(ctx, f)
		if err != nil {
			return fmt.Errorf("export employees: %w", err)
		}
		for i := range items {
			e := &items[i]
			row := []string{
				strconv.FormatInt(e.ID, 10),
				safeCell(e.Email),
				safeCell(e.FirstName),
				safeCell(e.LastName),
				safeCell(e.Phone),
				safeCell(e.DepartmentName),
				safeCell(e.PositionName),
				e.HireDate.Format("2006-01-02"),
				e.Status,
			}
			if err := cw.Write(row); err != nil {
				return fmt.Errorf("write row: %w", err)
			}
		}
		if len(items) == 0 || int64(f.Page*f.Limit) >= total {
			break
		}
		f.Page++
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func safeCell(v string) string {
	if v == "" {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		if !phoneLike(v) {
			return "'" + v
		}
	}
	return v
}

func phoneLike(v string) bool {
	hasDigit := false
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == '-' || r == '(' || r == ')' || r == ' ':
		default:
			return false
		}
	}
	return hasDigit
}
