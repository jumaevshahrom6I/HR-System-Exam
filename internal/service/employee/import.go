package employee

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"hr-system/internal/models"
)

const maxImportRows = 1000

var importColumns = []string{"email", "password", "first_name", "last_name", "phone", "department", "position", "hire_date"}

func (s *Service) Import(ctx context.Context, r io.Reader) (*models.ImportResult, error) {
	records, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("%w: file is empty", models.ErrInvalidInput)
	}

	columns := make(map[string]int, len(records[0]))
	for i, name := range records[0] {
		columns[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, name := range importColumns {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("%w: missing column %s", models.ErrInvalidInput, name)
		}
	}
	if len(records)-1 > maxImportRows {
		return nil, fmt.Errorf("%w: too many rows, max %d", models.ErrInvalidInput, maxImportRows)
	}

	refs := newRefCache(s)
	result := &models.ImportResult{Errors: []models.ImportRowError{}}
	for i, rec := range records[1:] {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("import interrupted: %w", err)
		}
		line := i + 2
		if err := s.importRow(ctx, rec, columns, refs); err != nil {
			result.Errors = append(result.Errors, models.ImportRowError{Row: line, Error: rowMessage(err)})
			continue
		}
		result.Created++
	}
	return result, nil
}

func readCSV(r io.Reader) ([][]string, error) {
	br := bufio.NewReader(r)
	if head, err := br.Peek(3); err == nil && bytes.Equal(head, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = br.Discard(3)
	}

	reader := csv.NewReader(br)
	reader.FieldsPerRecord = -1
	reader.Comma = detectComma(br)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid csv file", models.ErrInvalidInput)
	}
	return records, nil
}

func detectComma(br *bufio.Reader) rune {
	head, _ := br.Peek(4096)
	if i := bytes.IndexByte(head, '\n'); i >= 0 {
		head = head[:i]
	}
	if bytes.Count(head, []byte(";")) > bytes.Count(head, []byte(",")) {
		return ';'
	}
	return ','
}

func (s *Service) importRow(ctx context.Context, rec []string, columns map[string]int, refs *refCache) error {
	for _, i := range columns {
		if i >= len(rec) {
			return fmt.Errorf("%w: wrong number of columns", models.ErrInvalidInput)
		}
	}
	get := func(name string) string { return rec[columns[name]] }

	hireDate, err := time.Parse("2006-01-02", strings.TrimSpace(get("hire_date")))
	if err != nil {
		return fmt.Errorf("%w: hire_date must be in YYYY-MM-DD format", models.ErrInvalidInput)
	}
	departmentID, err := refs.department(ctx, get("department"))
	if err != nil {
		return err
	}
	positionID, err := refs.position(ctx, get("position"))
	if err != nil {
		return err
	}

	_, err = s.Create(ctx, models.EmployeeInput{
		Email:        get("email"),
		Password:     get("password"),
		FirstName:    get("first_name"),
		LastName:     get("last_name"),
		Phone:        get("phone"),
		DepartmentID: departmentID,
		PositionID:   positionID,
		HireDate:     hireDate,
	})
	return err
}

func rowMessage(err error) string {
	switch {
	case errors.Is(err, models.ErrConflict):
		return "email already registered"
	case errors.Is(err, models.ErrInvalidInput):
		return models.Message(err, models.ErrInvalidInput)
	default:
		return "internal error"
	}
}

type refCache struct {
	svc         *Service
	departments map[string]int64
	positions   map[string]int64
}

func newRefCache(s *Service) *refCache {
	return &refCache{svc: s, departments: map[string]int64{}, positions: map[string]int64{}}
}

func (c *refCache) department(ctx context.Context, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%w: department is required", models.ErrInvalidInput)
	}
	if id, ok := c.departments[name]; ok {
		return id, nil
	}
	d, err := c.svc.departments.GetByName(ctx, name)
	if errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("%w: department not found: %s", models.ErrInvalidInput, name)
	}
	if err != nil {
		return 0, fmt.Errorf("get department by name: %w", err)
	}
	c.departments[name] = d.ID
	return d.ID, nil
}

func (c *refCache) position(ctx context.Context, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%w: position is required", models.ErrInvalidInput)
	}
	if id, ok := c.positions[name]; ok {
		return id, nil
	}
	p, err := c.svc.positions.GetByName(ctx, name)
	if errors.Is(err, models.ErrNotFound) {
		return 0, fmt.Errorf("%w: position not found: %s", models.ErrInvalidInput, name)
	}
	if err != nil {
		return 0, fmt.Errorf("get position by name: %w", err)
	}
	c.positions[name] = p.ID
	return p.ID, nil
}
