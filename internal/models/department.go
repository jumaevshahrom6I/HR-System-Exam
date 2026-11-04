package models

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

type Department struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DepartmentInput struct {
	Name string
}

func (in *DepartmentInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(in.Name); n < 1 || n > 100 {
		return fmt.Errorf("%w: name must be 1-100 characters", ErrInvalidInput)
	}
	return nil
}
