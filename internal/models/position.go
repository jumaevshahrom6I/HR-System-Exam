package models

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Position struct {
	ID         int64
	Name       string
	BaseSalary float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PositionInput struct {
	Name       string
	BaseSalary float64
}

func (in *PositionInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > 100 {
		return fmt.Errorf("%w: name must be 1-100 characters", ErrInvalidInput)
	}
	if in.BaseSalary < 0 {
		return fmt.Errorf("%w: base_salary must be >= 0", ErrInvalidInput)
	}
	return nil
}
